/**
 * TrackLab - Gestion centralisée de l'état
 * Validation et persistance locale des données sportives.
 */

export const EVENTS = Object.freeze({
    "100m": "100 m",
    "200m": "200 m",
    "400m": "400 m",
    "100m_hurdles": "100 m haies",
    "110m_hurdles": "110 m haies",
    "400m_hurdles": "400 m haies",
    "long_jump": "Saut en longueur",
    "triple_jump": "Triple saut",
    "high_jump": "Saut en hauteur",
    "pole_vault": "Saut à la perche",
    "javelin": "Lancer du javelot",
    "discus": "Lancer du disque",
    "shot_put": "Lancer du poids",
    "hammer": "Lancer du marteau",
    "800m": "800 m",
    "1500m": "1 500 m",
    "3000m": "3 000 m",
    "5000m": "5 000 m",
    "10000m": "10 000 m",
    "half_marathon": "Semi-marathon",
    "marathon": "Marathon"
});

export const OBJECTIVES = Object.freeze({
    performance: "Performance",
    technique: "Technique",
    resistance: "Résistance",
    competition: "Compétition",
    recovery: "Récupération"
});

export const LEVELS = Object.freeze({
    excellent: "Excellent",
    good: "Favorable",
    moderate: "Modéré",
    poor: "Défavorable",
    very_poor: "Très défavorable",
    unavailable: "Indisponible"
});

const KEYS = Object.freeze({
    profile: "tracklab.profile.v1",
    sessions: "tracklab.sessions.v1",
    training: "tracklab.training.v1"
});

const MAX_SESSIONS = 5000;

const TRAINING_FIELDS = Object.freeze({
    sleep_hours: [0, 24],
    sleep_score: [0, 100],
    body_battery: [0, 100],
    resting_heart_rate: [1, 250],
    baseline_heart_rate: [1, 250],
    hrv: [1, 500],
    baseline_hrv: [1, 500],
    fatigue: [0, 10]
});

function readStorage(key, fallback) {
    try {
        const value = localStorage.getItem(key);

        return value === null
            ? fallback
            : JSON.parse(value);

    } catch {
        return fallback;
    }
}

function validNumber(value, minimum, maximum) {
    return typeof value === "number" &&
        Number.isFinite(value) &&
        value >= minimum &&
        value <= maximum;
}

function validDate(value) {
    if (
        typeof value !== "string" ||
        !/^\d{4}-\d{2}-\d{2}$/.test(value)
    ) {
        return false;
    }

    const date = new Date(`${value}T12:00:00`);

    return !Number.isNaN(date.getTime()) &&
        localDate(date) === value;
}

function normalizeCity(city) {
    if (!city || typeof city !== "object") {
        return null;
    }

    if (
        typeof city.name !== "string" ||
        !city.name.trim() ||
        !validNumber(city.latitude, -90, 90) ||
        !validNumber(city.longitude, -180, 180)
    ) {
        return null;
    }

    return {
        id: String(city.id ?? ""),
        name: city.name.trim().slice(0, 150),
        country: String(city.country ?? "").slice(0, 100),
        department: String(city.department ?? "").slice(0, 100),
        latitude: city.latitude,
        longitude: city.longitude
    };
}

function normalizeProfile(profile) {
    const value = profile && typeof profile === "object"
        ? profile
        : {};

    return {
        name: typeof value.name === "string"
            ? value.name.trim().slice(0, 60)
            : "",

        event: Object.hasOwn(EVENTS, value.event)
            ? value.event
            : "100m",

        objective: Object.hasOwn(OBJECTIVES, value.objective)
            ? value.objective
            : "performance",

        trackDirection: validNumber(
            value.trackDirection,
            0,
            360
        )
            ? value.trackDirection % 360
            : 0,

        city: normalizeCity(value.city)
    };
}

function validSession(session) {
    return Boolean(
        session &&
        typeof session.id === "string" &&
        session.id.trim() &&
        validDate(session.date) &&
        validNumber(session.duration_minutes, 1, 1440) &&
        validNumber(session.rpe, 0, 10)
    );
}

function normalizeSessions(sessions) {
    if (!Array.isArray(sessions)) {
        return [];
    }

    const identifiers = new Set();
    const normalized = [];

    for (const session of sessions) {
        if (
            !validSession(session) ||
            identifiers.has(session.id)
        ) {
            continue;
        }

        identifiers.add(session.id);

        normalized.push({
            id: session.id,
            date: session.date,
            duration_minutes: session.duration_minutes,
            rpe: session.rpe
        });

        if (normalized.length >= MAX_SESSIONS) {
            break;
        }
    }

    return normalized;
}

function normalizeTraining(training) {
    if (
        !training ||
        typeof training !== "object" ||
        Array.isArray(training)
    ) {
        return {};
    }

    const normalized = {};

    for (const [key, limits] of Object.entries(TRAINING_FIELDS)) {
        const value = training[key];

        if (value === null) {
            normalized[key] = null;
            continue;
        }

        if (validNumber(value, limits[0], limits[1])) {
            normalized[key] = value;
        }
    }

    return normalized;
}

const state = {
    profile: normalizeProfile({}),
    sessions: [],
    training: normalizeTraining({}),

    environment: null,
    analysis: null,
    garmin: null,
    health: null
};

const listeners = new Set();
let binding = null;
let writes = Promise.resolve();
let closing = false;
const savedKeys = new Set();

export function accountSessionHeaders() {
    return { "X-TrackLab-Session": binding ?? "" };
}

export function loadAccount(account) {
    binding = account.binding;
    const data = account.data ?? {};
    state.profile = normalizeProfile(data.profile);
    state.sessions = normalizeSessions(data.sessions);
    state.training = normalizeTraining(data.training);
    Object.keys(data).forEach(key => savedKeys.add(key));
}

export async function importLocalData() {
    let imported = false;
    for (const key of ["profile", "sessions"]) {
        if (savedKeys.has(key)) continue;
        const value = readStorage(KEYS[key], null);
        if (value === null) continue;
        imported = await Store.set(key, value, { onlyIfMissing: true }) || imported;
    }
    return imported;
}

export async function logoutAccount() {
    closing = true;
    try {
        await writes;
        const response = await fetch("/api/account/logout", {
            method: "POST",
            headers: { "X-TrackLab-Session": binding }
        });
        if (!response.ok) throw new Error("Déconnexion indisponible ou session modifiée. Actualisez la page.");
    } catch (error) {
        closing = false;
        throw error;
    }
}

function notify(key) {
    for (const listener of listeners) {
        try {
            listener(key);
        } catch {
            console.error(
                "TrackLab : erreur de mise à jour de l'interface."
            );
        }
    }
}

export const Store = Object.freeze({

    get(key) {
        return structuredClone(state[key]);
    },

    set(key, value, { onlyIfMissing = false } = {}) {
        if (!Object.hasOwn(state, key)) {
            throw new Error("État inconnu.");
        }

        if (key === "profile") {
            value = normalizeProfile(value);
        }

        if (key === "sessions") {
            if (
                !Array.isArray(value) ||
                value.length > MAX_SESSIONS
            ) {
                throw new Error(
                    "La limite de stockage des séances est atteinte."
                );
            }

            value = normalizeSessions(value);
        }

        if (key === "training") {
            value = normalizeTraining(value);
        }

        const snapshot = structuredClone(value);

        if (Object.hasOwn(KEYS, key)) {
            if (closing || !binding) return Promise.reject(new Error("Connexion requise."));
            const save = writes.then(async () => {
                const response = await fetch(`/api/account/data/${key}`, {
                    method: "PUT",
                    headers: {
                        "Content-Type": "application/json",
                        "X-TrackLab-Session": binding,
                        ...(onlyIfMissing ? { "If-None-Match": "*" } : {})
                    },
                    body: JSON.stringify(snapshot)
                });
                if (response.status === 401) {
                    window.location.reload();
                    throw new Error("La session a changé.");
                }
                if (onlyIfMissing && response.status === 412) return false;
                if (!response.ok) throw new Error("Enregistrement indisponible. Réessayez.");
                savedKeys.add(key);
                state[key] = snapshot;
                notify(key);
                return true;
            });
            writes = save.catch(() => {});
            return save;
        }

        state[key] = snapshot;

        notify(key);
    },

    updateProfile(changes) {
        return this.set("profile", {
            ...state.profile,
            ...changes
        });
    },

    addSession(session) {
        if (state.sessions.length >= MAX_SESSIONS) {
            throw new Error(
                "La limite de 5 000 séances est atteinte."
            );
        }

        if (
            !session ||
            !validDate(session.date) ||
            session.date > localDate() ||
            !validNumber(session.duration_minutes, 1, 1440) ||
            !validNumber(session.rpe, 0, 10)
        ) {
            throw new Error(
                "Les données de la séance sont invalides."
            );
        }

        return this.set("sessions", [
            {
                date: session.date,
                duration_minutes: session.duration_minutes,
                rpe: session.rpe,
                id: crypto.randomUUID()
            },
            ...state.sessions
        ]);
    },

    removeSession(id) {
        return this.set(
            "sessions",
            state.sessions.filter(
                session => session.id !== id
            )
        );
    },

    subscribe(listener) {
        if (typeof listener !== "function") {
            throw new Error("Abonnement invalide.");
        }

        listeners.add(listener);

        return () => listeners.delete(listener);
    }

});

export function localDate(date = new Date()) {
    const year = date.getFullYear();

    const month = String(
        date.getMonth() + 1
    ).padStart(2, "0");

    const day = String(
        date.getDate()
    ).padStart(2, "0");

    return `${year}-${month}-${day}`;
}

export function formatNumber(value, decimals = 1) {
    if (
        typeof value !== "number" ||
        !Number.isFinite(value)
    ) {
        return "Indisponible";
    }

    return new Intl.NumberFormat("fr-FR", {
        maximumFractionDigits: decimals
    }).format(value);
}

export function scoreColor(score) {
    if (score === null || !Number.isFinite(score)) {
        return "text-muted";
    }

    if (score >= 70) return "text-success";
    if (score >= 50) return "text-warning";

    return "text-danger";
}
