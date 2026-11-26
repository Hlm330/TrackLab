/**
 * TrackLab - Session Planner
 * Comparaison des créneaux d'entraînement.
 */

import { API } from "../api.js";

import {
    Store,
    EVENTS,
    OBJECTIVES,
    LEVELS,
    formatNumber,
    scoreColor,
} from "../store.js";

import {
    escapeHTML,
    selectOptions,
    renderRules,
    loading,
    errorState,
    icon
} from "../ui.js";

import {
    currentHour,
    environmentAt,
    sessionInput,
    signedNumber,
    formatHour
} from "../environment.js";

function formatDate(date) {
    return new Intl.DateTimeFormat("fr-FR", {
        weekday: "long",
        day: "numeric",
        month: "long"
    }).format(new Date(`${date}T12:00:00`));
}

async function mapConcurrent(items, limit, callback) {
    const results = new Array(items.length);
    let index = 0;

    async function worker() {
        while (index < items.length) {
            const current = index++;
            results[current] = await callback(items[current]);
        }
    }

    await Promise.all(
        Array.from(
            { length: Math.min(limit, items.length) },
            () => worker()
        )
    );

    return results;
}

function slotCard(slot, best) {
    const { conditions, result } = slot;
    const weather = conditions.weather;

    return `
        <article class="card slot-card stack ${best ? "best" : ""}">

            <div class="flex-row space-between wrap">
                <div>
                    <p class="metric-label">
                        ${escapeHTML(formatDate(conditions.time.slice(0, 10)))}
                    </p>

                    <h3 class="slot-time">
                        ${formatHour(conditions.time)}
                    </h3>
                </div>

                <strong class="slot-score ${scoreColor(result.score)}">
                    ${result.score}/100
                </strong>
            </div>

            ${best ? `
                <span class="badge badge-success">
                    Meilleur créneau analysé
                </span>
            ` : ""}

            <p class="${scoreColor(result.score)}">
                ${escapeHTML(LEVELS[result.level] || result.level)}
            </p>

            <div class="card-divider"></div>

            <div class="flex-row space-between">
                <span class="metric-label">Température</span>
                <strong>${formatNumber(weather.temperature)} °C</strong>
            </div>

            <div class="flex-row space-between">
                <span class="metric-label">Vent longitudinal</span>
                <strong>
                    ${signedNumber(result.wind.wind.longitudinal)} m/s
                </strong>
            </div>

            <div class="flex-row space-between">
                <span class="metric-label">Probabilité de pluie</span>
                <strong>
                    ${formatNumber(weather.precipitation_probability, 0)} %
                </strong>
            </div>

            <div class="flex-row space-between">
                <span class="metric-label">Qualité de l'air</span>
                <strong>
                    ${formatNumber(weather.air_quality_index, 0)} AQI
                </strong>
            </div>

            <p class="text-small text-muted">
                ${escapeHTML(result.recommendation)}
            </p>

            <details>
                <summary>Comprendre ce score</summary>
                ${renderRules(result.rules)}
            </details>

        </article>
    `;
}

function renderChart(canvas, slots) {
    if (typeof window.Chart !== "function") {
        canvas.hidden = true;
        return;
    }

    window.Chart.getChart(canvas)?.destroy();

    new window.Chart(canvas, {
        type: "line",

        data: {
            labels: slots.map(slot =>
                formatHour(slot.conditions.time)
            ),

            datasets: [
                {
                    label: "TrackLab Score",
                    data: slots.map(slot => slot.result.score),
                    borderColor: "#38BDF8",
                    backgroundColor: "rgba(56, 189, 248, 0.12)",
                    pointBackgroundColor: "#38BDF8",
                    pointRadius: 4,
                    tension: 0.3,
                    fill: true
                },
                {
                    label: "Score météo",
                    data: slots.map(slot => slot.result.weather.score),
                    borderColor: "#22C55E",
                    pointRadius: 2,
                    tension: 0.3
                },
                {
                    label: "Score vent",
                    data: slots.map(slot => slot.result.wind.score),
                    borderColor: "#F59E0B",
                    pointRadius: 2,
                    tension: 0.3
                }
            ]
        },

        options: {
            ...(window.matchMedia("(prefers-reduced-motion: reduce)").matches
                ? { animation: false }
                : {}),
            responsive: true,
            maintainAspectRatio: false,

            interaction: {
                mode: "index",
                intersect: false
            },

            scales: {
                y: {
                    min: 0,
                    max: 100,
                    ticks: {
                        color: "#94A3B8"
                    },
                    grid: {
                        color: "rgba(148, 163, 184, 0.1)"
                    }
                },

                x: {
                    ticks: {
                        color: "#94A3B8"
                    },
                    grid: {
                        display: false
                    }
                }
            },

            plugins: {
                legend: {
                    labels: {
                        color: "#E2E8F0",
                        usePointStyle: true
                    }
                }
            }
        }
    });
}

export function renderPlanner(container) {
    const profile = Store.get("profile");
    const environment = Store.get("environment");

    const now = currentHour(environment);

    const times = environment.weather.hourly.time.filter(
        time => time >= now
    );

    const dates = [...new Set(
        times.map(time => time.slice(0, 10))
    )];

    container.innerHTML = `
        <article class="card stack">

            <header class="card-header">
                <div>
                    <h2 class="card-title">
                        Planifier une séance
                    </h2>

                    <p class="card-description">
                        Comparez les conditions environnementales
                        pour trouver le créneau le plus adapté.
                    </p>
                </div>

                <span class="metric-icon">${icon("calendar")}</span>
            </header>

            <form id="planner-form" class="form-grid">

                <div class="form-group">
                    <label for="planner-event">Épreuve</label>

                    <select id="planner-event">
                        ${selectOptions(EVENTS, profile.event)}
                    </select>
                </div>

                <div class="form-group">
                    <label for="planner-objective">
                        Objectif de la séance
                    </label>

                    <select id="planner-objective">
                        ${selectOptions(OBJECTIVES, profile.objective)}
                    </select>
                </div>

                <div class="form-group full-width">
                    <label for="planner-date">Date</label>

                    <select id="planner-date">
                        ${dates.map(date => `
                            <option value="${escapeHTML(date)}">
                                ${escapeHTML(formatDate(date))}
                            </option>
                        `).join("")}
                    </select>
                </div>

                <div class="form-group">
                    <label for="planner-start">Heure de début</label>

                    <select id="planner-start">
                        ${Array.from({ length: 24 }, (_, hour) => `
                            <option value="${hour}"
                                ${hour === 8 ? "selected" : ""}>
                                ${String(hour).padStart(2, "0")}:00
                            </option>
                        `).join("")}
                    </select>
                </div>

                <div class="form-group">
                    <label for="planner-end">Heure de fin</label>

                    <select id="planner-end">
                        ${Array.from({ length: 24 }, (_, hour) => `
                            <option value="${hour}"
                                ${hour === 21 ? "selected" : ""}>
                                ${String(hour).padStart(2, "0")}:00
                            </option>
                        `).join("")}
                    </select>
                </div>

                <div class="form-group full-width">
                    <p class="form-help">
                        Ville : ${escapeHTML(profile.city.name)}.
                        Orientation de référence :
                        ${profile.trackDirection}°.
                    </p>

                    <p class="form-help">
                        Modifiez l'orientation du stade dans WindLab.
                    </p>
                </div>

                <div class="form-actions full-width">
                    <button type="submit"
                            id="planner-submit"
                            class="button button-primary">
                        ${icon("calendar")}
                        Comparer les créneaux
                    </button>
                </div>

            </form>

        </article>

        <div id="planner-results"></div>

        <div class="notice">
            <div>
                <strong>À propos des prévisions</strong>
                <p>
                    Les scores sont calculés pour chaque heure.
                    Ils ne garantissent pas les conditions pendant
                    toute la durée d'une séance.
                </p>
                <p>
                    La comparaison repose sur les données
                    environnementales uniquement.
                    La récupération future ne peut pas être prédite.
                </p>
                <p>
                    L'analyse du vent utilise la direction
                    de référence du profil.
                </p>
            </div>
        </div>
    `;

    const form = container.querySelector("#planner-form");
    const output = container.querySelector("#planner-results");
    const submit = container.querySelector("#planner-submit");

    async function analyze() {
        const event = container.querySelector("#planner-event").value;
        const objective = container.querySelector("#planner-objective").value;
        const date = container.querySelector("#planner-date").value;

        const start = Number(
            container.querySelector("#planner-start").value
        );

        const end = Number(
            container.querySelector("#planner-end").value
        );

        const previousCanvas = output.querySelector("canvas");

        if (previousCanvas && window.Chart) {
            window.Chart.getChart(previousCanvas)?.destroy();
        }

        if (start > end) {
            output.innerHTML = errorState(
                "L'heure de fin doit être supérieure ou égale à l'heure de début."
            );

            return;
        }

        const selectedTimes = times.filter(time => {
            const hour = Number(time.slice(11, 13));

            return time.startsWith(date) &&
                hour >= start &&
                hour <= end;
        });

        if (!selectedTimes.length) {
            output.innerHTML = errorState(
                "Aucun créneau disponible sur cette période."
            );

            return;
        }

        submit.disabled = true;

        output.innerHTML = loading(
            `Analyse de ${selectedTimes.length} créneaux...`
        );

        try {
            const analyses = await mapConcurrent(
                selectedTimes,
                4,
                async time => {
                    try {
                        const conditions = environmentAt(
                            environment,
                            time
                        );

                        const result = await API.analyzeSession(
                            sessionInput(
                                {
                                    ...profile,
                                    event,
                                    objective
                                },
                                conditions,
                                {}
                            )
                        );

                        return { conditions, result };

                    } catch {
                        return null;
                    }
                }
            );

            if (!output.isConnected) return;

            const slots = analyses.filter(Boolean);

            if (!slots.length) {
                throw new Error(
                    "Aucun créneau n'a pu être analysé."
                );
            }

            const ranked = [...slots].sort((a, b) =>
                b.result.score - a.result.score ||
                a.conditions.time.localeCompare(b.conditions.time)
            );

            const best = ranked[0];

            const unavailable = analyses.length - slots.length;

            output.innerHTML = `
                <div class="stack">

                    ${unavailable ? `
                        <div class="notice notice-warning">
                            ${unavailable} créneau(x) indisponible(s)
                            n'ont pas été intégrés au classement.
                        </div>
                    ` : ""}

                    <article class="card stack">

                        <div class="flex-row space-between wrap">
                            <div>
                                <span class="badge badge-success">
                                    Meilleur créneau
                                </span>

                                <h2 class="score-number">
                                    ${formatHour(best.conditions.time)}
                                </h2>

                                <p class="text-muted">
                                    ${escapeHTML(formatDate(date))}
                                </p>
                            </div>

                            <strong class="score-number
                                ${scoreColor(best.result.score)}">
                                ${best.result.score}<small>/100</small>
                            </strong>
                        </div>

                        <p>
                            ${escapeHTML(best.result.recommendation)}
                        </p>

                        <p class="text-small text-muted">
                            ${slots.length} créneaux comparés.
                            En cas d'égalité, le premier créneau
                            chronologique est présenté.
                        </p>

                    </article>

                    <article class="card stack">

                        <h2 class="card-title">
                            Évolution des conditions
                        </h2>

                        <p class="card-description">
                            Comparaison horaire des scores
                            environnementaux.
                        </p>

                        <div class="chart-container">
                            <canvas id="planner-chart"></canvas>
                        </div>

                    </article>

                    <h2 class="section-title">
                        Classement des créneaux
                    </h2>

                    <div class="grid grid-3">
                        ${ranked.map((slot, index) =>
                            slotCard(slot, index === 0)
                        ).join("")}
                    </div>

                </div>
            `;

            renderChart(
                output.querySelector("#planner-chart"),
                slots
            );

        } catch (error) {
            output.innerHTML = errorState(error.message);

        } finally {
            submit.disabled = false;
        }
    }

    form.addEventListener("submit", event => {
        event.preventDefault();
        analyze();
    });

    analyze();
}
