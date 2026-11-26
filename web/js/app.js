/**
 * TrackLab - Application principale
 * Navigation, initialisation et coordination des modules.
 */

import { API } from "./api.js";
import { Store, importLocalData, loadAccount, logoutAccount } from "./store.js";

import {
    initializeIcons,
    loading,
    empty,
    errorState,
    toast
} from "./ui.js";

import { renderDashboard } from "./modules/dashboard.js";
import { renderWindLab } from "./modules/windlab.js";
import { renderPlanner } from "./modules/planner.js";
import { renderTraining } from "./modules/training.js";
import { renderDataHub } from "./modules/datahub.js";
import { renderProfile } from "./modules/profile.js";

const PAGES = Object.freeze({
    dashboard: "Dashboard",
    windlab: "WindLab",
    planner: "Session Planner",
    training: "Training Lab",
    datahub: "Data Hub",
    profile: "Profil sportif"
});

const RENDERERS = Object.freeze({
    dashboard: renderDashboard,
    windlab: renderWindLab,
    planner: renderPlanner,
    training: renderTraining,
    datahub: renderDataHub,
    profile: renderProfile
});

const ENVIRONMENT_PAGES = new Set([
    "dashboard",
    "windlab",
    "planner"
]);

const ENVIRONMENT_TTL = 5 * 60 * 1000;

let currentPage = null;
let navigationVersion = 0;
let serverCheckVersion = 0;

const pendingEnvironment = new Map();

function updateHeader() {
    const profile = Store.get("profile");

    const location = document.getElementById("header-location");
    const avatar = document.getElementById("header-avatar");

    location.textContent =
        profile.city?.name || "Choisir une ville";

    const initials = profile.name
        .split(/\s+/)
        .filter(Boolean)
        .slice(0, 2)
        .map(word => word[0].toUpperCase())
        .join("");

    avatar.textContent = initials || "TL";
}

async function checkServer() {
    const version = ++serverCheckVersion;

    const status = document.getElementById("server-status");
    const dot = document.getElementById("server-status-dot");

    try {
        const health = await API.health();

        if (health.status !== "ok") {
            throw new Error("Serveur indisponible.");
        }

        if (version !== serverCheckVersion) return;

        Store.set("health", health);

        status.textContent = "Serveur opérationnel";
        dot.className = "status-dot online";

    } catch {
        if (version !== serverCheckVersion) return;

        Store.set("health", null);

        status.textContent = "Serveur indisponible";
        dot.className = "status-dot offline";
    }
}

function closeMenu() {
    const sidebar = document.getElementById("sidebar");
    const overlay = document.getElementById("sidebar-overlay");
    const toggle = document.getElementById("menu-toggle");

    sidebar.classList.remove("open");
    overlay.hidden = true;

    toggle.setAttribute("aria-expanded", "false");
}

function initializeMenu() {
    const sidebar = document.getElementById("sidebar");
    const overlay = document.getElementById("sidebar-overlay");
    const toggle = document.getElementById("menu-toggle");

    toggle.addEventListener("click", () => {
        const open = sidebar.classList.toggle("open");

        overlay.hidden = !open;

        toggle.setAttribute(
            "aria-expanded",
            String(open)
        );
    });

    overlay.addEventListener("click", closeMenu);

    document.addEventListener("keydown", event => {
        if (event.key === "Escape") {
            closeMenu();
        }
    });
}

function destroyCharts(container) {
    if (typeof window.Chart !== "function") return;

    container.querySelectorAll("canvas").forEach(canvas => {
        window.Chart.getChart(canvas)?.destroy();
    });
}

async function refreshEnvironment(force = false) {
    const profile = Store.get("profile");

    if (!profile.city) {
        Store.set("environment", null);
        return null;
    }

    const { latitude, longitude } = profile.city;

    const cached = Store.get("environment");

    if (
        !force &&
        cached &&
        cached.latitude === latitude &&
        cached.longitude === longitude &&
        Date.now() - cached.fetchedAt < ENVIRONMENT_TTL
    ) {
        return cached;
    }

    const key = `${latitude}:${longitude}`;

    if (pendingEnvironment.has(key)) {
        return pendingEnvironment.get(key);
    }

    const request = (async () => {
        try {
            const environment = await API.environment(
                latitude,
                longitude
            );

            const result = {
                ...environment,
                latitude,
                longitude
            };

            const selectedCity = Store.get("profile").city;

            if (
                selectedCity?.latitude === latitude &&
                selectedCity?.longitude === longitude
            ) {
                Store.set("environment", result);
            }

            return result;

        } finally {
            pendingEnvironment.delete(key);
        }
    })();

    pendingEnvironment.set(key, request);

    return request;
}

function updateNavigation(page) {
    document.querySelectorAll(".page").forEach(section => {
        section.hidden = section.id !== `page-${page}`;
    });

    document.querySelectorAll(".nav-link").forEach(link => {
        const active = link.dataset.page === page;

        link.classList.toggle("active", active);

        if (active) {
            link.setAttribute("aria-current", "page");
        } else {
            link.removeAttribute("aria-current");
        }
    });

    document.getElementById("breadcrumb-page").textContent =
        PAGES[page];

    document.title = `${PAGES[page]} | TrackLab`;

    closeMenu();
    updateHeader();
}

async function navigate(page, force = false) {
    if (!Object.hasOwn(PAGES, page)) {
        page = "dashboard";
    }

    const version = ++navigationVersion;

    if (currentPage) {
        const previous = document.getElementById(
            `${currentPage}-content`
        );

        if (previous) {
            destroyCharts(previous);
            previous.replaceChildren();
        }
    }

    currentPage = page;

    updateNavigation(page);

    const container = document.getElementById(
        `${page}-content`
    );

    if (!container) return;

    /*
     * Un conteneur indépendant empêche une ancienne requête
     * de modifier une page après un changement de navigation.
     */

    const view = document.createElement("div");

    view.className = "stack";

    container.replaceChildren(view);

    try {
        if (ENVIRONMENT_PAGES.has(page)) {
            const profile = Store.get("profile");

            if (!profile.city) {
                view.innerHTML = empty(
                    "Configurez votre espace sportif",
                    "Sélectionnez votre ville et votre épreuve pour accéder aux analyses personnalisées.",
                    '<a class="button button-primary" href="#profile">Configurer mon profil</a>'
                );

                return;
            }

            view.innerHTML = loading(
                "Récupération des données environnementales..."
            );

            await refreshEnvironment(force);

            if (
                version !== navigationVersion ||
                !view.isConnected
            ) return;
        }

        const render = RENDERERS[page];

        await render(view);

    } catch (error) {
        if (
            version === navigationVersion &&
            view.isConnected
        ) {
            view.innerHTML = errorState(error.message);
        }
    }
}

function pageFromHash() {
    const page = window.location.hash.slice(1);

    return Object.hasOwn(PAGES, page)
        ? page
        : "dashboard";
}

function initializeRefresh() {
    const button = document.getElementById("refresh-button");

    button.addEventListener("click", async () => {
        button.disabled = true;

        try {
            await checkServer();

            await navigate(
                currentPage || "dashboard",
                true
            );

        } finally {
            button.disabled = false;
        }
    });
}

function initialize() {
    initializeIcons();
    initializeMenu();
    initializeRefresh();
    updateHeader();

    Store.subscribe(key => {
        if (key === "profile") {
            updateHeader();
            Store.set("analysis", null);
        }
    });

    window.addEventListener("hashchange", () => {
        if (window.location.hash === "#main-content") {
            return;
        }

        navigate(pageFromHash());

        window.scrollTo({
            top: 0
        });
    });

    checkServer();

    navigate(pageFromHash());

    window.setInterval(
        checkServer,
        60000
    );
}

async function initializeAccount() {
    const accountChannel = typeof BroadcastChannel === "function"
        ? new BroadcastChannel("tracklab-account")
        : null;
    accountChannel?.addEventListener("message", () => window.location.reload());
    const screen = document.getElementById("account-screen");
    const app = document.getElementById("account-app");
    const form = document.getElementById("account-form");
    const status = document.getElementById("account-status");
    const email = document.getElementById("account-email");
    const password = document.getElementById("account-password");
    const submit = document.getElementById("account-submit");
    const toggle = document.getElementById("account-toggle");
    const logout = document.getElementById("account-logout");
    const importButton = document.getElementById("account-import");
    let creating = false;

    const outdatedServer = "Le serveur TrackLab utilise une ancienne version. Redémarre-le, puis actualise cette page.";

    function open(account) {
        loadAccount(account);
        form.reset();
        screen.hidden = true;
        app.hidden = false;
        importButton.hidden = ["profile", "sessions"].every(key => Object.hasOwn(account.data ?? {}, key));
        initialize();
        document.getElementById("main-content").focus();
    }

    form.addEventListener("submit", async event => {
        event.preventDefault();
        submit.disabled = toggle.disabled = true;
        status.textContent = "Connexion en cours…";
        try {
            const response = await fetch(creating ? "/api/account/register" : "/api/account/login", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ email: email.value, password: password.value })
            });
            if (response.status === 404 || response.status === 405) {
                status.textContent = outdatedServer;
                return;
            }
            if (!response.headers.get("Content-Type")?.includes("application/json")) {
                status.textContent = "Réponse inattendue du serveur TrackLab. Actualise la page et réessaie.";
                return;
            }
            const account = await response.json();
            if (!response.ok) throw new Error(account.error || "Connexion indisponible.");
            open(account);
            accountChannel?.postMessage("account-changed");
        } catch (error) {
            status.textContent = error.message;
        } finally {
            submit.disabled = toggle.disabled = false;
        }
    });

    toggle.addEventListener("click", () => {
        creating = !creating;
        document.getElementById("account-title").textContent = creating ? "Créer un compte TrackLab" : "Connexion à TrackLab";
        submit.textContent = creating ? "Créer mon compte" : "Se connecter";
        toggle.textContent = creating ? "Revenir à la connexion" : "Créer un compte";
        password.autocomplete = creating ? "new-password" : "current-password";
        password.minLength = creating ? 12 : 0;
        status.textContent = creating ? "Choisissez un mot de passe d’au moins 12 caractères." : "";
    });

    logout.addEventListener("click", async () => {
        logout.disabled = true;
        try {
            await logoutAccount();
            accountChannel?.postMessage("account-changed");
            app.hidden = true;
            window.location.replace("/");
        } catch (error) {
            toast(error.message, "error");
            logout.disabled = false;
        }
    });

    importButton.addEventListener("click", async () => {
        if (!window.confirm("Copier le profil et les séances de ce navigateur qui ne sont pas encore enregistrés dans ce compte ?")) return;
        importButton.disabled = true;
        try {
            const imported = await importLocalData();
            if (imported) await navigate(currentPage || "dashboard");
            toast(imported ? "Données locales importées." : "Aucune donnée locale à importer.", imported ? "success" : "");
            importButton.hidden = true;
        } catch (error) {
            toast(error.message, "error");
        } finally {
            importButton.disabled = false;
        }
    });

    try {
        const response = await fetch("/api/account", { cache: "no-store" });
        if (response.ok) {
            open(await response.json());
            return;
        }
        if (response.status === 404 || response.status === 405) {
            status.textContent = outdatedServer;
            return;
        }
        status.textContent = response.status === 401 ? "" : "Vérification de la session indisponible.";
    } catch {
        status.textContent = "Le serveur est indisponible.";
    }
    form.hidden = false;
}

window.addEventListener("pageshow", event => {
    if (event.persisted) window.location.reload();
});

initializeAccount();
