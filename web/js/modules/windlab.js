/**
 * TrackLab - WindLab
 * Analyse du vent par zone d'entraînement.
 */

import { API } from "../api.js";
import { Store, EVENTS, OBJECTIVES, formatNumber, scoreColor } from "../store.js";
import { escapeHTML, renderRules, icon, loading, errorState, toast } from "../ui.js";
import { currentHour, environmentAt, signedNumber } from "../environment.js";

const ZONES = [
    { id: "main", name: "Ligne droite principale", angles: [90] },
    { id: "opposite", name: "Ligne droite opposée", angles: [270] },
    { id: "left", name: "Virage gauche", angles: [240, 210, 180, 150, 120] },
    { id: "right", name: "Virage droit", angles: [60, 30, 0, 330, 300] },
    { id: "jumpTop", name: "Longueur / Triple saut (haut)", angles: [90] },
    { id: "jumpBottom", name: "Longueur / Triple saut (bas)", angles: [90] },
    { id: "poleLeft", name: "Perche (gauche)", angles: [0] },
    { id: "poleRight", name: "Perche (droite)", angles: [0] },
    { id: "highLeft", name: "Hauteur (gauche)", angles: [90] },
    { id: "highRight", name: "Hauteur (droite)", angles: [270] }
];

const normalize = angle => ((angle % 360) + 360) % 360;

function defaultZones(event) {
    switch (event) {
        case "100m":
        case "100m_hurdles":
        case "110m_hurdles":
            return ["main"];

        case "200m":
            return ["left", "main"];

        case "long_jump":
        case "triple_jump":
            return ["jumpBottom"];

        case "pole_vault":
            return ["poleRight"];

        case "high_jump":
            return ["highLeft"];

        default:
            return ["main", "opposite", "left", "right"];
    }
}

function stadiumSVG(selected, rotation, directions, windDirection) {
    const color = id => selected.has(id) ? "#38BDF8" : "#99463E";
    const runway = id => selected.has(id) ? "#38BDF8" : "#B7B7AD";

    const lanes = Array.from({ length: 9 }, (_, i) => `
        <rect
            x="${24 + i * 3.75}"
            y="${25 + i * 3.75}"
            width="${472 - i * 7.5}"
            height="${280 - i * 7.5}"
            rx="${140 - i * 3.75}"
            stroke="#E6BEB5"
            stroke-width=".7"
            fill="none"
            pointer-events="none"
        />
    `).join("");

    const sprintLanes = Array.from({ length: 9 }, (_, i) => `
        <path
            d="M79 ${275 + i * 3.75}H441"
            stroke="#F8FAFC"
            stroke-width=".7"
            pointer-events="none"
        />
    `).join("");

    const jumpZones = [
        { id: "jumpTop", y: 68 },
        { id: "jumpBottom", y: 261 }
    ].map(({ id, y }) => `
        <g data-zone="${id}" class="wind-zone">
            <rect x="153" y="${y - 13}" width="214" height="26"
                  fill="transparent"/>
            <rect x="177" y="${y - 4}" width="158" height="8"
                  rx="1" fill="${runway(id)}"/>
            <rect x="335" y="${y - 9}" width="24" height="18"
                  fill="#D6B78B"/>
            <rect x="153" y="${y - 9}" width="24" height="18"
                  fill="#D6B78B"/>

            ${selected.has(id) ? `
                <g transform="rotate(${directions[id] || 0} 256 ${y})">
                    <path d="M195 ${y}H317"
                          stroke="white" stroke-width="2"
                          marker-end="url(#running-arrow)"/>
                </g>
            ` : ""}
        </g>
    `).join("");

    const poleZones = [
        { id: "poleLeft", x: 124 },
        { id: "poleRight", x: 396 }
    ].map(({ id, x }) => `
        <g data-zone="${id}" class="wind-zone">
            <rect x="${x - 17}" y="80" width="34" height="170"
                  fill="transparent"/>
            <rect x="${x - 4}" y="102" width="8" height="126"
                  fill="${runway(id)}"/>
            <rect x="${x - 15}" y="82" width="30" height="20"
                  rx="2" fill="#2563EB"/>
            <rect x="${x - 15}" y="228" width="30" height="20"
                  rx="2" fill="#2563EB"/>

            ${selected.has(id) ? `
                <g transform="rotate(${directions[id] || 0} ${x} 165)">
                    <path d="M${x} 214V116"
                          stroke="white" stroke-width="2"
                          marker-end="url(#running-arrow)"/>
                </g>
            ` : ""}
        </g>
    `).join("");

    const highZones = [
        { id: "highLeft", x: 84 },
        { id: "highRight", x: 436 }
    ].map(({ id, x }) => `
        <g data-zone="${id}" class="wind-zone">
            <circle cx="${x}" cy="165" r="21"
                    fill="${selected.has(id) ? "#38BDF8" : "#85524A"}"
                    stroke="#C6A39C" stroke-dasharray="3 3"/>
            <rect x="${x - 13}" y="151" width="26" height="13"
                  rx="2" fill="#2563EB"/>
            <path d="M${x - 13} 167h26"
                  stroke="#F8FAFC" stroke-width="1.5"/>

            ${selected.has(id) ? `
                <g transform="rotate(${(id === "highRight" ? 180 : 0) + (directions[id] || 0)} ${x} 165)">
                    <path d="M${x - 16} 165H${x + 16}"
                          stroke="white" stroke-width="2"
                          marker-end="url(#running-arrow)"/>
                </g>
            ` : ""}
        </g>
    `).join("");

    return `
        <svg viewBox="0 0 640 640"
             xmlns="http://www.w3.org/2000/svg"
             role="img"
             aria-label="Stade d'athlétisme interactif">

            <defs>
                <marker id="running-arrow"
                        markerWidth="5" markerHeight="5"
                        refX="4" refY="2.5" orient="auto">
                    <path d="M0 0L5 2.5L0 5Z" fill="white"/>
                </marker>

                <marker id="wind-arrow"
                        markerWidth="6" markerHeight="6"
                        refX="5" refY="3" orient="auto">
                    <path d="M0 0L6 3L0 6Z" fill="#F59E0B"/>
                </marker>
            </defs>

            <rect width="640" height="640" rx="18" fill="#111B2D"/>

            <circle cx="320" cy="320" r="290"
                    stroke="#334155" stroke-dasharray="5 8"
                    fill="none"/>

            <g fill="#94A3B8" font-size="15" text-anchor="middle">
                <text x="320" y="22">N</text>
                <text x="622" y="325">E</text>
                <text x="320" y="628">S</text>
                <text x="18" y="325">O</text>
            </g>

            <g transform="translate(60 155)">
                <g transform="rotate(${rotation} 260 165)">

                    <rect x="24" y="25" width="472" height="280"
                          rx="140" fill="#99463E"/>

                    <rect x="54" y="55" width="412" height="220"
                          rx="110" fill="#733E37"/>

                    <rect x="147" y="55" width="226" height="220"
                          fill="#28573D"/>

                    <path data-zone="opposite" class="wind-zone"
                          d="M164 40H356"
                          stroke="${color("opposite")}"
                          stroke-width="30"/>

                    <path data-zone="left" class="wind-zone"
                          d="M164 40A125 125 0 0 0 164 290"
                          stroke="${color("left")}"
                          stroke-width="30" fill="none"/>

                    <path data-zone="right" class="wind-zone"
                          d="M356 40A125 125 0 0 1 356 290"
                          stroke="${color("right")}"
                          stroke-width="30" fill="none"/>

                    ${lanes}

                    <rect data-zone="main" class="wind-zone"
                          x="79" y="275" width="362" height="30"
                          fill="${color("main")}"/>

                    ${sprintLanes}

                    <path d="M79 275V305M441 275V305"
                          stroke="white" stroke-width="1.5"
                          pointer-events="none"/>

                    <g stroke="#8EAF96" stroke-width=".8"
                       fill="none" pointer-events="none">

                        <rect x="154" y="87" width="212" height="156"/>
                        <path d="M260 87V243"/>
                        <circle cx="260" cy="165" r="19"/>
                        <path d="M154 132H188V198H154
                                 M366 132H332V198H366"/>
                    </g>

                    ${jumpZones}
                    ${poleZones}
                    ${highZones}

                    <g fill="white" font-size="10"
                       text-anchor="middle" pointer-events="none">

                        <text x="260" y="44">LIGNE DROITE OPPOSÉE</text>
                        <text x="260" y="294">LIGNE DROITE PRINCIPALE</text>

                        <text x="40" y="165"
                              transform="rotate(-90 40 165)">VIRAGE</text>

                        <text x="480" y="165"
                              transform="rotate(90 480 165)">VIRAGE</text>
                    </g>

                    <g stroke="white" stroke-width="2"
                       fill="none" pointer-events="none">

                        <path d="M310 48H210"
                              marker-end="url(#running-arrow)"/>

                        <path d="M210 283H310"
                              marker-end="url(#running-arrow)"/>
                    </g>
                </g>
            </g>

            <g transform="rotate(${windDirection} 320 320)"
               stroke="#F59E0B" stroke-width="3" fill="none"
               pointer-events="none">

                <path d="M280 42V100" marker-end="url(#wind-arrow)"/>
                <path d="M320 42V100" marker-end="url(#wind-arrow)"/>
                <path d="M360 42V100" marker-end="url(#wind-arrow)"/>
            </g>

        </svg>
    `;
}

function resultCard(zone, results) {
    const scores = results.map(result => result.score);
    const winds = results.map(result => result.wind);

    const average = Math.round(
        scores.reduce((sum, score) => sum + score, 0) / scores.length
    );

    const longitudinal = winds.map(wind => wind.longitudinal);
    const lateral = winds.map(wind => wind.crosswind);

    const min = Math.min(...longitudinal);
    const max = Math.max(...longitudinal);

    const representative = results.reduce(
        (worst, result) => result.score < worst.score ? result : worst
    );

    const windLabel = results.length === 1
        ? `${signedNumber(min)} m/s`
        : `${signedNumber(min)} à ${signedNumber(max)} m/s`;

    const regulated = results.length === 1 &&
        representative.regulation.applicable;

    return `
        <article class="card stack">
            <div class="flex-row space-between">
                <h3 class="card-title">${escapeHTML(zone.name)}</h3>

                <strong class="metric-value ${scoreColor(average)}">
                    ${average}/100
                </strong>
            </div>

            <div class="flex-row space-between">
                <span class="metric-label">Vent longitudinal</span>
                <strong>${windLabel}</strong>
            </div>

            <div class="flex-row space-between">
                <span class="metric-label">Vent latéral maximal</span>
                <strong>${formatNumber(Math.max(...lateral))} m/s</strong>
            </div>

            <p class="text-small text-muted">
                ${escapeHTML(representative.performance_effect)}
            </p>

            <p class="text-small text-muted">
                ${escapeHTML(representative.technical_effect)}
            </p>

            ${regulated ? `
                <div class="notice ${representative.regulation.exceeded
                    ? "notice-warning" : ""}">
                    ${escapeHTML(representative.regulation.message)}
                </div>
            ` : ""}

            ${results.length > 1 ? `
                <p class="text-small text-muted">
                    Score moyen de ${results.length} orientations
                    échantillonnées dans le virage.
                </p>
            ` : ""}

            ${renderRules(representative.rules.filter(rule =>
                regulated || rule.category !== "regulation"
            ))}
        </article>
    `;
}

export function renderWindLab(container) {
    const profile = Store.get("profile");
    const environment = Store.get("environment");

    const conditions = environmentAt(
        environment,
        currentHour(environment)
    );

    let rotation = normalize(profile.trackDirection - 90);
    let selected = new Set(defaultZones(profile.event));
    let directions = {};
    let version = 0;

    container.innerHTML = `
        <div class="grid grid-main">
            <article class="card stack">

                <header class="card-header">
                    <div>
                        <h2 class="card-title">Votre stade</h2>
                        <p class="card-description">
                            Sélectionnez directement les zones utilisées.
                        </p>
                    </div>
                    <span class="metric-icon">${icon("wind")}</span>
                </header>

                <div id="wind-stadium" class="wind-visualization"></div>

                <div class="flex-row wrap">
                    <span class="badge badge-primary">Zone sélectionnée</span>
                    <span class="badge badge-warning">Direction du vent</span>
                </div>

                <div class="form-group">
                    <label for="stadium-rotation">
                        Orientation géographique du stade
                    </label>

                    <input type="range" id="stadium-rotation"
                           min="0" max="359" step="1"
                           value="${rotation}"/>

                    <p id="stadium-angle" class="form-help"></p>

                    <p class="form-help">
                        Alignez le schéma sur l'orientation réelle
                        de votre stade.
                    </p>
                </div>

                <button id="save-orientation"
                        class="button button-secondary">
                    Enregistrer cette orientation
                </button>

            </article>

            <article class="card stack">

                <h2 class="card-title">Configuration de la séance</h2>

                <p class="text-muted">
                    ${escapeHTML(EVENTS[profile.event])} ·
                    ${escapeHTML(OBJECTIVES[profile.objective])}
                </p>

                <div class="flex-row space-between">
                    <span class="metric-label">Vitesse du vent</span>
                    <strong>${formatNumber(conditions.wind.speed)} km/h</strong>
                </div>

                <div class="flex-row space-between">
                    <span class="metric-label">Provenance du vent</span>
                    <strong>${conditions.wind.direction}°</strong>
                </div>

                <div id="wind-zone-list" class="stack"></div>

                <button id="wind-select-all"
                        class="button button-secondary">
                    Sélectionner l'anneau complet
                </button>

                <button id="wind-clear"
                        class="button button-secondary">
                    Tout désélectionner
                </button>

                <button id="wind-analyze"
                        class="button button-primary button-block">
                    Analyser les zones
                </button>

            </article>
        </div>

        <h2 class="section-title">Analyse par portion</h2>

        <div id="wind-results" class="grid grid-2"></div>

        <div class="notice">
            <div>
                <strong>Limites de l'analyse</strong>
                <p>
                    Le stade est schématique. Les emplacements réels
                    des aires de saut peuvent différer.
                </p>
                <p>
                    Les virages sont analysés par échantillonnage
                    de plusieurs directions. Les scores ne constituent
                    pas des prédictions chronométriques.
                </p>
                <p>
                    Le vent provient des prévisions Open-Meteo à 10 m
                    et ne remplace pas un anémomètre réglementaire.
                </p>
            </div>
        </div>
    `;

    const stadium = container.querySelector("#wind-stadium");
    const zoneList = container.querySelector("#wind-zone-list");
    const results = container.querySelector("#wind-results");
    const angleLabel = container.querySelector("#stadium-angle");
    const analyzeButton = container.querySelector("#wind-analyze");

    function refreshStadium() {
        stadium.innerHTML = stadiumSVG(
            selected,
            rotation,
            directions,
            conditions.wind.direction
        );

        angleLabel.textContent =
            `Direction de la ligne droite principale : ${normalize(rotation + 90)}°`;
    }

    function refreshSelection() {
        refreshStadium();

        zoneList.innerHTML = ZONES.map(zone => `
            <div class="wind-zone-option">
                <label class="flex-row">
                    <input type="checkbox"
                           data-select-zone="${zone.id}"
                           ${selected.has(zone.id) ? "checked" : ""}/>

                    <span>${escapeHTML(zone.name)}</span>
                </label>

                ${selected.has(zone.id) &&
                    /^(jump|pole|high)/.test(zone.id) ? `

                    <div class="form-group">
                        <label for="direction-${zone.id}">
                            ${zone.id.startsWith("high")
                                ? "Direction d'approche"
                                : "Sens d'élan"}
                        </label>

                        <select id="direction-${zone.id}"
                                data-direction-zone="${zone.id}">

                            ${(zone.id.startsWith("high")
                                ? [0, 45, 90, 135, 180, 225, 270, 315]
                                : [0, 180]).map(angle => `
                                    <option value="${angle}"
                                        ${(directions[zone.id] || 0) === angle
                                            ? "selected" : ""}>
                                        ${zone.id.startsWith("high")
                                            ? `${angle}° relatif au schéma`
                                            : angle === 0
                                                ? "Sens initial"
                                                : "Sens inverse"}
                                    </option>
                                `).join("")}

                        </select>
                    </div>
                ` : ""}
            </div>
        `).join("");

        analyzeButton.disabled = selected.size === 0;
    }

    function invalidateResults() {
        version++;

        results.innerHTML = `
            <article class="card">
                <p class="text-muted">
                    ${selected.size
                        ? "Configuration modifiée. Lancez une nouvelle analyse."
                        : "Sélectionnez au moins une zone."}
                </p>
            </article>
        `;

        analyzeButton.disabled = selected.size === 0;
    }

    function toggleZone(id) {
        if (selected.has(id)) {
            selected.delete(id);
        } else {
            selected.add(id);
        }

        refreshSelection();
        invalidateResults();
    }

    stadium.addEventListener("click", event => {
        const target = event.target.closest("[data-zone]");

        if (target) {
            toggleZone(target.dataset.zone);
        }
    });

    zoneList.addEventListener("change", event => {
        const target = event.target;

        if (target.dataset.selectZone) {
            toggleZone(target.dataset.selectZone);
        }

        if (target.dataset.directionZone) {
            directions[target.dataset.directionZone] = Number(target.value);
            refreshStadium();
            invalidateResults();
        }
    });

    container.querySelector("#stadium-rotation")
        .addEventListener("input", event => {
            rotation = Number(event.target.value);
            refreshStadium();
            invalidateResults();
        });

    container.querySelector("#save-orientation")
        .addEventListener("click", async event => {
            const button = event.currentTarget;
            button.disabled = true;
            try {
                await Store.updateProfile({ trackDirection: normalize(rotation + 90) });
                button.textContent = "Orientation enregistrée";
            } catch (error) {
                toast(error.message, "error");
            } finally {
                button.disabled = false;
            }
        });

    container.querySelector("#wind-select-all")
        .addEventListener("click", () => {
            selected = new Set(["main", "opposite", "left", "right"]);
            refreshSelection();
            invalidateResults();
        });

    container.querySelector("#wind-clear")
        .addEventListener("click", () => {
            selected.clear();
            refreshSelection();
            invalidateResults();
        });

    async function analyze() {
        const requestVersion = ++version;

        const activeZones = ZONES.filter(zone => selected.has(zone.id));

        if (!activeZones.length) return;

        analyzeButton.disabled = true;
        results.innerHTML = loading("Analyse des différentes zones...");

        try {
            const analyses = await Promise.all(
                activeZones.map(async zone => {
                    const zoneResults = await Promise.all(
                        zone.angles.map(angle =>
                            API.analyzeWind({
                                event: profile.event,
                                objective: profile.objective,
                                wind: {
                                    speed: conditions.wind.speed,
                                    direction: conditions.wind.direction,
                                    track_direction: normalize(
                                        angle + rotation +
                                        (directions[zone.id] || 0)
                                    )
                                }
                            })
                        )
                    );

                    return { zone, results: zoneResults };
                })
            );

            if (requestVersion !== version || !results.isConnected) return;

            results.innerHTML = analyses.map(analysis =>
                resultCard(analysis.zone, analysis.results)
            ).join("");

        } catch (error) {
            if (requestVersion === version) {
                results.innerHTML = errorState(error.message);
            }
        } finally {
            if (requestVersion === version) {
                analyzeButton.disabled = selected.size === 0;
            }
        }
    }

    analyzeButton.addEventListener("click", analyze);

    refreshSelection();
    analyze();
}
