/**
 * TrackLab - Training Lab
 * Préparation sportive et suivi de la charge d'entraînement.
 */

import { API } from "../api.js";

import {
    Store,
    LEVELS,
    localDate,
    formatNumber,
    scoreColor
} from "../store.js";

import {
    escapeHTML,
    metric,
    scoreRing,
    renderRules,
    loading,
    errorState,
    empty,
    toast,
    icon
} from "../ui.js";

const INDICATORS = [
    {
        key: "sleep_hours",
        label: "Durée du sommeil",
        unit: "heures",
        min: 0,
        max: 24,
        step: 0.1,
        placeholder: "8"
    },
    {
        key: "sleep_score",
        label: "Score de sommeil",
        unit: "/100",
        min: 0,
        max: 100,
        step: 1,
        placeholder: "85"
    },
    {
        key: "body_battery",
        label: "Body Battery",
        unit: "/100",
        min: 0,
        max: 100,
        step: 1,
        placeholder: "80"
    },
    {
        key: "resting_heart_rate",
        label: "Fréquence cardiaque au repos",
        unit: "bpm",
        min: 1,
        max: 250,
        step: 1,
        placeholder: "56"
    },
    {
        key: "baseline_heart_rate",
        label: "Fréquence cardiaque habituelle",
        unit: "bpm",
        min: 1,
        max: 250,
        step: 1,
        placeholder: "55"
    },
    {
        key: "hrv",
        label: "Variabilité cardiaque (VFC)",
        unit: "ms",
        min: 1,
        max: 500,
        step: 1,
        placeholder: "65"
    },
    {
        key: "baseline_hrv",
        label: "VFC habituelle",
        unit: "ms",
        min: 1,
        max: 500,
        step: 1,
        placeholder: "70"
    },
    {
        key: "fatigue",
        label: "Fatigue ressentie",
        unit: "/10",
        min: 0,
        max: 10,
        step: 1,
        placeholder: "3"
    }
];

function indicatorField(indicator, training) {
    const value = training[indicator.key] ?? "";

    return `
        <div class="form-group">
            <label for="training-${indicator.key}">
                ${escapeHTML(indicator.label)}
            </label>

            <input
                type="number"
                id="training-${indicator.key}"
                name="${indicator.key}"
                min="${indicator.min}"
                max="${indicator.max}"
                step="${indicator.step}"
                value="${escapeHTML(value)}"
                placeholder="${indicator.placeholder}"
            />

            <span class="form-help">
                ${escapeHTML(indicator.unit)}
            </span>
        </div>
    `;
}

function readIndicators(form) {
    const values = {};

    for (const indicator of INDICATORS) {
        const input = form.elements.namedItem(indicator.key);

        values[indicator.key] = input.value === ""
            ? null
            : Number(input.value);
    }

    return values;
}

function preparationResult(result) {
    if (!result.available) {
        return empty(
            "Préparation non évaluée",
            "Renseignez au moins un indicateur exploitable."
        );
    }

    return `
        <div class="stack">

            <div class="score-layout">
                <div>
                    <p class="metric-label">Score de préparation</p>

                    <div class="score-number ${scoreColor(result.score)}">
                        ${result.score}<small>/100</small>
                    </div>

                    <p class="score-label ${scoreColor(result.score)}">
                        ${escapeHTML(LEVELS[result.level] || result.level)}
                    </p>
                </div>

                ${scoreRing(result.score)}
            </div>

            <div class="card-divider"></div>

            <div class="flex-row space-between">
                <span class="metric-label">Couverture des données</span>
                <strong>${result.data_coverage} %</strong>
            </div>

            <progress
                value="${result.data_coverage}"
                max="100"
                aria-label="Couverture des indicateurs">
            </progress>

            <p class="text-small text-muted">
                ${result.available_indicators} indicateurs disponibles
                sur ${result.total_indicators}.
            </p>

            ${result.data_coverage < 60 ? `
                <div class="notice notice-warning">
                    Données limitées. Ce score doit être interprété
                    avec prudence.
                </div>
            ` : ""}

            ${renderRules(result.rules)}

        </div>
    `;
}

function displayDate(date) {
    const parsed = new Date(`${date}T12:00:00`);

    if (Number.isNaN(parsed.getTime())) {
        return "Date inconnue";
    }

    return new Intl.DateTimeFormat("fr-FR", {
        day: "numeric",
        month: "short",
        year: "numeric"
    }).format(parsed);
}

function dateBefore(days) {
    const date = new Date();

    date.setDate(date.getDate() - days);

    return localDate(date);
}

function renderLoadChart(canvas, sessions) {
    if (typeof window.Chart !== "function") {
        canvas.hidden = true;
        return;
    }

    window.Chart.getChart(canvas)?.destroy();

    const dates = Array.from(
        { length: 28 },
        (_, index) => dateBefore(27 - index)
    );

    const dailyLoads = new Map();

    for (const session of sessions) {
        const load = session.duration_minutes * session.rpe;

        dailyLoads.set(
            session.date,
            (dailyLoads.get(session.date) || 0) + load
        );
    }

    const values = dates.map(date => dailyLoads.get(date) || 0);

    new window.Chart(canvas, {
        type: "bar",

        data: {
            labels: dates.map(date => {
                const [, month, day] = date.split("-");
                return `${day}/${month}`;
            }),

            datasets: [
                {
                    label: "Charge quotidienne",
                    data: values,

                    backgroundColor: values.map(value =>
                        value > 0 ? "#38BDF8" : "#334155"
                    ),

                    borderRadius: 4
                }
            ]
        },

        options: {
            ...(window.matchMedia("(prefers-reduced-motion: reduce)").matches
                ? { animation: false }
                : {}),
            responsive: true,
            maintainAspectRatio: false,

            scales: {
                y: {
                    beginAtZero: true,

                    ticks: {
                        color: "#94A3B8"
                    },

                    grid: {
                        color: "rgba(148, 163, 184, 0.1)"
                    }
                },

                x: {
                    ticks: {
                        color: "#94A3B8",
                        maxTicksLimit: 14
                    },

                    grid: {
                        display: false
                    }
                }
            },

            plugins: {
                legend: {
                    labels: {
                        color: "#E2E8F0"
                    }
                }
            }
        }
    });
}

function sessionHistory(sessions) {
    if (!sessions.length) {
        return empty(
            "Aucune séance enregistrée",
            "Ajoutez votre première séance pour suivre votre charge."
        );
    }

    const sorted = [...sessions].sort((a, b) =>
        b.date.localeCompare(a.date)
    );

    return `
        <div class="table-container">

            <table class="data-table">

                <thead>
                    <tr>
                        <th>Date</th>
                        <th>Durée</th>
                        <th>RPE</th>
                        <th>Charge</th>
                        <th>Action</th>
                    </tr>
                </thead>

                <tbody>

                    ${sorted.map(session => `
                        <tr>
                            <td>
                                ${escapeHTML(displayDate(session.date))}
                            </td>

                            <td>
                                ${formatNumber(session.duration_minutes, 0)}
                                min
                            </td>

                            <td>
                                ${formatNumber(session.rpe)}/10
                            </td>

                            <td class="text-primary">
                                ${formatNumber(
                                    session.duration_minutes * session.rpe,
                                    0
                                )}
                                UA
                            </td>

                            <td>
                                <button
                                    type="button"
                                    class="button button-danger button-small"
                                    data-delete-session="${escapeHTML(session.id)}"
                                    aria-label="Supprimer cette séance">

                                    ${icon("trash")}

                                </button>
                            </td>
                        </tr>
                    `).join("")}

                </tbody>

            </table>

        </div>
    `;
}

export function renderTraining(container) {
    const training = Store.get("training");

    container.innerHTML = `
        <div class="grid grid-main">

            <article class="card stack">

                <header class="card-header">
                    <div>
                        <h2 class="card-title">Préparation sportive</h2>

                        <p class="card-description">
                            Renseignez vos indicateurs de récupération.
                        </p>
                    </div>

                    <span class="metric-icon">
                        ${icon("heart")}
                    </span>
                </header>

                <form id="preparation-form" class="form-grid">

                    ${INDICATORS.map(indicator =>
                        indicatorField(indicator, training)
                    ).join("")}

                    <div class="form-group full-width">
                        <p class="form-help">
                            Tous les champs sont facultatifs.
                            Laissez vide une donnée inconnue.
                        </p>

                        <p class="form-help">
                            Fatigue : 0 signifie aucune fatigue,
                            10 signifie fatigue maximale.
                        </p>
                    </div>

                    <div class="form-actions full-width">

                        <button type="submit"
                                id="preparation-submit"
                                class="button button-primary">

                            ${icon("activity")}
                            Analyser ma préparation

                        </button>

                        <button type="button"
                                id="preparation-clear"
                                class="button button-secondary">

                            Réinitialiser

                        </button>

                    </div>

                </form>

            </article>

            <article class="card stack">

                <h2 class="card-title">État de préparation</h2>

                <div id="preparation-result"></div>

            </article>

        </div>

        <h2 class="section-title">Charge d'entraînement</h2>

        <div id="training-load" class="grid grid-4"></div>

        <div class="grid grid-main">

            <article class="card stack">

                <header class="card-header">
                    <div>
                        <h2 class="card-title">
                            Évolution de la charge
                        </h2>

                        <p class="card-description">
                            Historique des 28 derniers jours.
                        </p>
                    </div>
                </header>

                <div class="chart-container">
                    <canvas id="training-chart"></canvas>
                </div>

            </article>

            <article class="card stack">

                <header class="card-header">
                    <div>
                        <h2 class="card-title">Ajouter une séance</h2>

                        <p class="card-description">
                            Calcul de la charge par la méthode session-RPE.
                        </p>
                    </div>

                    <span class="metric-icon">
                        ${icon("plus")}
                    </span>
                </header>

                <form id="session-form" class="stack">

                    <div class="form-group">
                        <label for="session-date">Date</label>

                        <input type="date"
                               id="session-date"
                               value="${localDate()}"
                               max="${localDate()}"
                               required/>
                    </div>

                    <div class="form-group">
                        <label for="session-duration">
                            Durée en minutes
                        </label>

                        <input type="number"
                               id="session-duration"
                               min="1"
                               max="1440"
                               step="1"
                               placeholder="90"
                               required/>
                    </div>

                    <div class="form-group">
                        <label for="session-rpe">
                            Intensité ressentie (RPE)
                        </label>

                        <select id="session-rpe">

                            ${[
                                "Repos",
                                "Très facile",
                                "Facile",
                                "Modéré",
                                "Assez difficile",
                                "Difficile",
                                "Soutenu",
                                "Très difficile",
                                "Intense",
                                "Très intense",
                                "Maximal"
                            ].map((label, index) => `
                                <option value="${index}"
                                    ${index === 5 ? "selected" : ""}>

                                    ${index}/10 · ${label}

                                </option>
                            `).join("")}

                        </select>
                    </div>

                    <div class="notice">
                        <div>
                            <span class="metric-label">
                                Charge estimée
                            </span>

                            <strong id="session-preview"
                                    class="metric-value text-primary">

                                0 UA

                            </strong>
                        </div>
                    </div>

                    <button type="submit"
                            class="button button-primary button-block">

                        ${icon("plus")}
                        Enregistrer la séance

                    </button>

                </form>

            </article>

        </div>

        <article class="card stack">

            <header class="card-header">
                <div>
                    <h2 class="card-title">Historique des séances</h2>

                    <p class="card-description">
                        Vos séances sont conservées localement
                        dans votre navigateur.
                    </p>
                </div>
            </header>

            <div id="session-history"></div>

        </article>

        <div class="notice">
            <div>
                <strong>Comprendre la charge d'entraînement</strong>

                <p>
                    Charge = durée en minutes × intensité ressentie.
                    Le résultat est exprimé en unités arbitraires (UA).
                </p>

                <p>
                    La charge enregistrée ne comprend que les séances
                    ajoutées dans TrackLab.
                </p>

                <p>
                    Les scores de préparation sont des indicateurs
                    heuristiques. Ils ne constituent ni un diagnostic
                    médical ni une prédiction du risque de blessure.
                </p>
            </div>
        </div>
    `;

    const preparationForm = container.querySelector("#preparation-form");
    const preparationOutput = container.querySelector("#preparation-result");
    const preparationSubmit = container.querySelector("#preparation-submit");

    const sessionForm = container.querySelector("#session-form");
    const durationInput = container.querySelector("#session-duration");
    const rpeInput = container.querySelector("#session-rpe");
    const sessionPreview = container.querySelector("#session-preview");

    const loadOutput = container.querySelector("#training-load");
    const historyOutput = container.querySelector("#session-history");
    const canvas = container.querySelector("#training-chart");

    let preparationVersion = 0;
    let loadVersion = 0;

    async function analyzePreparation(values, notify = false) {
        const version = ++preparationVersion;

        preparationSubmit.disabled = true;

        preparationOutput.innerHTML = loading(
            "Analyse de votre préparation..."
        );

        try {
            const result = await API.analyzeTraining(values);

            if (
                version !== preparationVersion ||
                !preparationOutput.isConnected
            ) return;

            if (notify) await Store.set("training", values);
            Store.set("analysis", null);

            preparationOutput.innerHTML = preparationResult(result);

            if (notify) {
                toast("Préparation sportive actualisée.");
            }

        } catch (error) {
            if (
                version === preparationVersion &&
                preparationOutput.isConnected
            ) {
                preparationOutput.innerHTML = errorState(error.message);
            }

        } finally {
            if (version === preparationVersion) {
                preparationSubmit.disabled = false;
            }
        }
    }

    async function refreshLoad() {
        const version = ++loadVersion;
        const sessions = Store.get("sessions");

        historyOutput.innerHTML = sessionHistory(sessions);

        loadOutput.innerHTML = loading(
            "Calcul de la charge d'entraînement..."
        );

        try {
            renderLoadChart(canvas, sessions);
        } catch {
            canvas.hidden = true;
        }

        try {
            const payload = sessions.map(session => ({
                date: session.date,
                duration_minutes: session.duration_minutes,
                rpe: session.rpe
            }));

            const result = await API.analyzeLoad(
                payload,
                localDate()
            );

            if (
                version !== loadVersion ||
                !loadOutput.isConnected
            ) return;

            loadOutput.innerHTML = `

                ${metric(
                    "Charge sur 7 jours",
                    formatNumber(result.load_7_days, 0),
                    "UA",
                    "activity",
                    `${result.sessions_7_days} séance(s)`
                )}

                ${metric(
                    "Charge sur 28 jours",
                    formatNumber(result.load_28_days, 0),
                    "UA",
                    "calendar",
                    `${result.sessions_28_days} séance(s)`
                )}

                ${metric(
                    "Moyenne hebdomadaire",
                    formatNumber(result.weekly_average, 0),
                    "UA",
                    "target",
                    "Charge sur 28 jours divisée par quatre"
                )}

                ${metric(
                    "Séances enregistrées",
                    sessions.length,
                    "",
                    "database",
                    "Historique local"
                )}

            `;

        } catch (error) {
            if (
                version === loadVersion &&
                loadOutput.isConnected
            ) {
                loadOutput.innerHTML = errorState(error.message);
            }
        }
    }

    function updatePreview() {
        const duration = Number(durationInput.value);
        const rpe = Number(rpeInput.value);

        const load = Number.isFinite(duration) &&
            duration >= 1 &&
            duration <= 1440
            ? duration * rpe
            : 0;

        sessionPreview.textContent =
            `${formatNumber(load, 0)} UA`;
    }

    preparationForm.addEventListener("submit", event => {
        event.preventDefault();

        analyzePreparation(
            readIndicators(preparationForm),
            true
        );
    });

    container.querySelector("#preparation-clear")
        .addEventListener("click", () => {
            for (const indicator of INDICATORS) {
                preparationForm.elements
                    .namedItem(indicator.key).value = "";
            }

            analyzePreparation({}, true);
        });

    durationInput.addEventListener("input", updatePreview);
    rpeInput.addEventListener("change", updatePreview);

    sessionForm.addEventListener("submit", async event => {
        event.preventDefault();

        const date = container.querySelector("#session-date").value;
        const duration = Number(durationInput.value);
        const rpe = Number(rpeInput.value);

        if (
            !date ||
            date > localDate() ||
            !Number.isFinite(duration) ||
            duration < 1 ||
            duration > 1440 ||
            !Number.isFinite(rpe) ||
            rpe < 0 ||
            rpe > 10
        ) {
            toast("Les données de la séance sont invalides.", "error");
            return;
        }

        const submit = sessionForm.querySelector('button[type="submit"]');
        if (submit.disabled || sessionForm.inert) return;
        submit.disabled = true;
        historyOutput.inert = true;
        try {
            await Store.addSession({
                date,
                duration_minutes: duration,
                rpe
            });

            durationInput.value = "";
            updatePreview();

            toast("Séance enregistrée.", "success");

            if (window.matchMedia("(prefers-reduced-motion: no-preference)").matches) {
                const buttonIcon = submit.querySelector("svg");
                buttonIcon?.animate?.(
                    [{ scale: "1" }, { scale: "1.24" }, { scale: "1" }],
                    { duration: 340, easing: "ease-in-out" }
                );
            }

            refreshLoad();

        } catch (error) {
            toast(error.message, "error");
        } finally {
            submit.disabled = false;
            historyOutput.inert = false;
        }
    });

    historyOutput.addEventListener("click", async event => {
        const button = event.target.closest("[data-delete-session]");

        if (!button || historyOutput.inert) return;

        if (!window.confirm("Supprimer définitivement cette séance ?")) {
            return;
        }

        historyOutput.inert = sessionForm.inert = true;
        try {
            await Store.removeSession(button.dataset.deleteSession);

            toast("Séance supprimée.");

            refreshLoad();

        } catch (error) {
            toast(error.message, "error");
        } finally {
            historyOutput.inert = sessionForm.inert = false;
        }
    });

    analyzePreparation(training);
    refreshLoad();
}
