/**
 * TrackLab - Composants réutilisables
 */

import { formatNumber, scoreColor } from "./store.js";

const ICONS = Object.freeze({
    dashboard: '<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/>',

    wind: '<path d="M3 8h12a3 3 0 1 0-3-3M3 12h16a3 3 0 1 1-3 3M3 16h7a3 3 0 1 1-3 3"/>',

    calendar: '<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M16 3v4M8 3v4M3 10h18"/>',

    activity: '<path d="M3 12h4l3-8 4 16 3-8h4"/>',

    database: '<ellipse cx="12" cy="5" rx="8" ry="3"/><path d="M4 5v14c0 4 16 4 16 0V5M4 12c0 4 16 4 16 0"/>',

    user: '<circle cx="12" cy="8" r="4"/><path d="M4 21v-2a8 8 0 0 1 16 0v2"/>',

    location: '<path d="M20 10c0 6-8 12-8 12S4 16 4 10a8 8 0 1 1 16 0Z"/><circle cx="12" cy="10" r="2.5"/>',

    refresh: '<path d="M20 7v5h-5M4 17v-5h5M20 12a8 8 0 0 0-14-5M4 12a8 8 0 0 0 14 5"/>',

    menu: '<path d="M4 6h16M4 12h16M4 18h16"/>',

    thermometer: '<path d="M9 14V5a3 3 0 0 1 6 0v9a5 5 0 1 1-6 0Z"/><path d="M12 9v9"/>',

    droplets: '<path d="M12 3C9 8 5 12 5 15a7 7 0 0 0 14 0c0-3-4-7-7-12Z"/>',

    battery: '<rect x="2" y="6" width="18" height="12" rx="2"/><path d="M22 10v4M6 10h10v4H6Z"/>',

    sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M2 12h2M20 12h2M5 5l2 2M17 17l2 2M19 5l-2 2M7 17l-2 2"/>',

    rain: '<path d="M5 14a4 4 0 1 1 2-7 6 6 0 0 1 11 3 3 3 0 1 1 0 6H5M8 18l-1 3M13 18l-1 3M18 18l-1 3"/>',

    heart: '<path d="M12 21 4 13C-2 7 7 0 12 7c5-7 14 0 8 6Z"/>',

    check: '<path d="m5 12 4 4L19 6"/>',

    warning: '<path d="M12 3 2 21h20Z"/><path d="M12 9v5M12 17h.01"/>',

    target: '<circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="5"/><circle cx="12" cy="12" r="1"/>',

    clock: '<circle cx="12" cy="12" r="9"/><path d="M12 6v6l4 2"/>',

    trash: '<path d="M3 6h18M9 6V3h6v3M6 6l1 15h10l1-15M10 10v7M14 10v7"/>',

    plus: '<path d="M12 5v14M5 12h14"/>'
});

export function icon(name) {
    if (!Object.hasOwn(ICONS, name)) return "";

    return `<svg xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.7"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true">${ICONS[name]}</svg>`;
}

export function initializeIcons(root = document) {
    root.querySelectorAll("[data-icon]").forEach(element => {
        element.innerHTML = icon(element.dataset.icon);
    });
}

export function escapeHTML(value) {
    return String(value ?? "").replace(/[&<>"']/g, character => ({
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        '"': "&quot;",
        "'": "&#39;"
    })[character]);
}

export function loading(message = "Chargement des données...") {
    return `
        <div class="loading-state" role="status">
            <span class="spinner"></span>
            <p>${escapeHTML(message)}</p>
        </div>
    `;
}

export function empty(title, message, action = "") {
    return `
        <div class="empty-state">
            <h3>${escapeHTML(title)}</h3>
            <p>${escapeHTML(message)}</p>
            ${action}
        </div>
    `;
}

export function errorState(message) {
    return `
        <div class="error-state" role="alert">
            <span class="metric-icon text-danger">${icon("warning")}</span>
            <h3>Données indisponibles</h3>
            <p>${escapeHTML(message)}</p>
        </div>
    `;
}

export function toast(message, type = "") {
    const container = document.getElementById("toast-container");
    const element = document.createElement("div");

    element.className = "toast";

    if (["success", "error"].includes(type)) {
        element.classList.add(type);
    }

    element.textContent = message;
    container.append(element);

    setTimeout(() => element.remove(), 4500);
}

export function metric(label, value, unit, iconName, caption = "") {
    return `
        <article class="card metric-card">
            <span class="metric-icon">${icon(iconName)}</span>
            <span class="metric-label">${escapeHTML(label)}</span>
            <div class="metric-value">
                ${escapeHTML(value)}
                <span class="metric-unit">${escapeHTML(unit)}</span>
            </div>
            <span class="metric-caption">${escapeHTML(caption)}</span>
        </article>
    `;
}

export function scoreRing(score) {
    const available = typeof score === "number" &&
        Number.isFinite(score);

    const value = available
        ? Math.min(100, Math.max(0, score))
        : 0;

    return `
        <svg class="score-ring ${scoreColor(score)}"
             viewBox="0 0 120 120"
             role="img"
             aria-label="${available ? value + " sur 100" : "Score indisponible"}">
            <circle cx="60" cy="60" r="48"
                    fill="none" stroke="#29374C" stroke-width="9"/>
            <circle class="score-ring-progress" cx="60" cy="60" r="48"
                    pathLength="100" fill="none"
                    stroke="${value > 0 ? "currentColor" : "none"}" stroke-width="9"
                    stroke-dasharray="${value} 100"
                    stroke-linecap="round"
                    transform="rotate(-90 60 60)"/>
            <text x="60" y="65" fill="currentColor"
                  text-anchor="middle" font-size="25" font-weight="700">
                ${available ? formatNumber(value, 0) : "N/D"}
            </text>
        </svg>
    `;
}

export function renderRules(rules = []) {
    if (!Array.isArray(rules) || rules.length === 0) {
        return `
            <div class="notice">
                Aucune contrainte particulière détectée par les règles d'analyse.
            </div>
        `;
    }

    return `
        <div class="rule-list">
            ${rules.map(rule => {
                const severity = ["low", "moderate", "high"]
                    .includes(rule.severity)
                    ? rule.severity
                    : "low";

                const regulation = rule.category === "regulation";

                return `
                    <article class="rule-item ${severity} ${regulation ? "regulation" : ""}">
                        <div>
                            <span class="badge ${regulation ? "badge-primary" : ""}">
                                ${regulation ? "Réglementation" : "Analyse sportive"}
                            </span>
                            <h3>${escapeHTML(rule.message)}</h3>
                            <p>${escapeHTML(rule.recommendation)}</p>
                            ${rule.penalty > 0
                                ? `<p>Pénalité : ${escapeHTML(rule.penalty)} points</p>`
                                : ""}
                        </div>
                    </article>
                `;
            }).join("")}
        </div>
    `;
}

export function selectOptions(options, selected) {
    return Object.entries(options).map(([value, label]) => `
        <option value="${escapeHTML(value)}"
            ${value === selected ? "selected" : ""}>
            ${escapeHTML(label)}
        </option>
    `).join("");
}
