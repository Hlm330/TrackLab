/**
 * TrackLab - Tableau de bord
 */

import { API } from "../api.js";

import {
    Store,
    EVENTS,
    OBJECTIVES,
    LEVELS,
    formatNumber,
    scoreColor
} from "../store.js";

import {
    escapeHTML,
    metric,
    scoreRing,
    renderRules,
    loading,
    icon
} from "../ui.js";

import {
    currentHour,
    environmentAt,
    sessionInput,
    airQualityLabel,
    cardinalDirection,
    signedNumber
} from "../environment.js";

function breakdown(label, score, weight) {
    return `
        <div>
            <p class="metric-label">${escapeHTML(label)}</p>

            <strong class="metric-value ${scoreColor(score)}">
                ${score === null ? "N/D" : formatNumber(score, 0)}
            </strong>

            <p class="metric-caption">
                Pondération : ${formatNumber(weight * 100, 0)} %
            </p>
        </div>
    `;
}

export async function renderDashboard(container) {
    const profile = Store.get("profile");
    const environment = Store.get("environment");
    const training = Store.get("training");

    container.innerHTML = loading("Calcul du TrackLab Score...");

    const time = currentHour(environment);
    const conditions = environmentAt(environment, time);

    const result = await API.analyzeSession(
        sessionInput(profile, conditions, training)
    );

    Store.set("analysis", result);

    const weather = conditions.weather;
    const wind = result.wind.wind;

    const greeting = profile.name
        ? `Bonjour ${profile.name}`
        : "Bienvenue sur TrackLab";

    container.innerHTML = `

        <article class="card">
            <div class="flex-row space-between wrap">
                <div>
                    <h2 class="section-title">
                        ${escapeHTML(greeting)}
                    </h2>

                    <p class="card-description">
                        ${escapeHTML(profile.city.name)} ·
                        ${escapeHTML(EVENTS[profile.event])} ·
                        ${escapeHTML(OBJECTIVES[profile.objective])}
                    </p>
                </div>

                <a href="#profile" class="button button-secondary">
                    ${icon("user")}
                    Modifier mon profil
                </a>
            </div>
        </article>

        <div class="grid grid-main">

            <article class="card score-card">

                <header class="card-header">
                    <div>
                        <span class="eyebrow">ANALYSE PERSONNALISÉE</span>
                        <h2 class="card-title">TrackLab Score</h2>
                    </div>

                    <span class="badge badge-primary">
                        ${result.environmental_only
                            ? "Analyse environnementale"
                            : "Analyse combinée"}
                    </span>
                </header>

                <div class="score-layout">

                    <div>
                        <div class="score-number ${scoreColor(result.score)}">
                            ${result.score}<small>/100</small>
                        </div>

                        <p class="score-label ${scoreColor(result.score)}">
                            ${escapeHTML(LEVELS[result.level] || result.level)}
                        </p>

                        <p class="score-recommendation">
                            ${escapeHTML(result.recommendation)}
                        </p>
                    </div>

                    ${scoreRing(result.score)}

                </div>

                <div class="score-breakdown">

                    ${breakdown(
                        "Météo",
                        result.weather.score,
                        result.weights.weather
                    )}

                    ${breakdown(
                        "Vent",
                        result.wind.score,
                        result.weights.wind
                    )}

                    ${breakdown(
                        "Préparation",
                        result.training.score,
                        result.weights.training
                    )}

                </div>

            </article>

            <article class="card stack">

                <header class="card-header">
                    <div>
                        <h2 class="card-title">Préparation sportive</h2>
                        <p class="card-description">
                            Disponibilité des indicateurs personnels
                        </p>
                    </div>

                    <span class="metric-icon">${icon("activity")}</span>
                </header>

                <div>
                    <span class="metric-label">Couverture des données</span>

                    <div class="metric-value text-primary">
                        ${result.training.data_coverage} %
                    </div>

                    <p class="metric-caption">
                        ${result.training.available_indicators}
                        indicateurs sur
                        ${result.training.total_indicators}
                    </p>
                </div>

                <progress
                    value="${result.training.data_coverage}"
                    max="100"
                    aria-label="Couverture des données personnelles">
                </progress>

                <p class="text-muted text-small">
                    ${result.environmental_only
                        ? "Aucune donnée personnelle disponible. Le score repose uniquement sur les conditions environnementales."
                        : "Les indicateurs disponibles contribuent au score selon leur couverture."}
                </p>

                <a href="#training"
                   class="button button-primary button-block">
                    Accéder à Training Lab
                </a>

            </article>

        </div>

        <div class="flex-row space-between wrap">
            <h2 class="section-title">Conditions environnementales</h2>

            <span class="badge">
                ${escapeHTML(time.replace("T", " à "))}
            </span>
        </div>

        <div class="grid grid-4">

            ${metric(
                "Température",
                formatNumber(weather.temperature),
                "°C",
                "thermometer",
                `Ressenti : ${formatNumber(weather.apparent_temperature)} °C`
            )}

            ${metric(
                "Vent longitudinal",
                signedNumber(wind.longitudinal),
                "m/s",
                "wind",
                wind.longitudinal_type === "tailwind"
                    ? "Vent favorable"
                    : wind.longitudinal_type === "headwind"
                        ? "Vent de face"
                        : "Vent longitudinal neutre"
            )}

            ${metric(
                "Humidité",
                formatNumber(weather.humidity, 0),
                "%",
                "droplets",
                "Humidité relative"
            )}

            ${metric(
                "Qualité de l'air",
                formatNumber(weather.air_quality_index, 0),
                "AQI",
                "activity",
                airQualityLabel(weather.air_quality_index)
            )}

            ${metric(
                "Vitesse du vent",
                formatNumber(conditions.wind.speed),
                "km/h",
                "wind",
                `Provenance : ${cardinalDirection(conditions.wind.direction)}`
            )}

            ${metric(
                "Rafales",
                formatNumber(weather.wind_gusts),
                "km/h",
                "wind",
                "Rafales prévues à 10 mètres"
            )}

            ${metric(
                "Probabilité de pluie",
                formatNumber(weather.precipitation_probability, 0),
                "%",
                "rain",
                `${formatNumber(weather.precipitation)} mm de précipitations`
            )}

            ${metric(
                "Indice UV",
                formatNumber(weather.uv_index),
                "",
                "sun",
                "Exposition aux ultraviolets"
            )}

        </div>

        <div class="grid grid-2">

            <article class="card stack">
                <h2 class="card-title">Analyse du vent</h2>

                <div class="flex-row space-between">
                    <span class="metric-label">Vent longitudinal</span>
                    <strong>${signedNumber(wind.longitudinal)} m/s</strong>
                </div>

                <div class="flex-row space-between">
                    <span class="metric-label">Vent latéral</span>
                    <strong>${formatNumber(wind.crosswind)} m/s</strong>
                </div>

                <p class="text-small text-muted">
                    ${escapeHTML(result.wind.performance_effect)}
                </p>

                <p class="text-small text-muted">
                    ${escapeHTML(result.wind.technical_effect)}
                </p>

                <div class="notice ${result.wind.regulation.exceeded
                    ? "notice-warning"
                    : ""}">
                    ${escapeHTML(result.wind.regulation.message)}
                </div>

                <a href="#windlab" class="button button-secondary">
                    Ouvrir WindLab
                </a>

            </article>

            <article class="card stack">

                <h2 class="card-title">Planifier ma prochaine séance</h2>

                <span class="metric-icon">${icon("calendar")}</span>

                <p class="text-muted">
                    Comparez les prochaines heures pour identifier
                    les conditions les plus adaptées à votre épreuve
                    et à votre objectif sportif.
                </p>

                <a href="#planner" class="button button-primary">
                    Ouvrir Session Planner
                </a>

            </article>

        </div>

        <article class="card">

            <header class="card-header">
                <div>
                    <h2 class="card-title">Analyse et recommandations</h2>
                    <p class="card-description">
                        Explication des règles ayant influencé votre score.
                    </p>
                </div>
            </header>

            ${renderRules(result.rules)}

        </article>

        <div class="notice">
            <div>
                <strong>Sources des données</strong>
                <p>
                    Open-Meteo Forecast et Air Quality.
                    Données horaires modélisées, et non mesures effectuées
                    directement sur votre piste.
                </p>
                <p>
                    Les scores TrackLab reposent sur des règles heuristiques
                    d'aide à la décision.
                </p>
            </div>
        </div>
    `;
}
