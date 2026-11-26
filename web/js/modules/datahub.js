/**
 * TrackLab - Data Hub
 * Synchronisation et exploitation des données Garmin.
 */

import { API } from "../api.js";

import {
    Store,
    localDate,
    formatNumber
} from "../store.js";

import {
    escapeHTML,
    metric,
    loading,
    empty,
    errorState,
    toast,
    icon
} from "../ui.js";

function numeric(...values) {
    return values.find(value =>
        typeof value === "number" &&
        Number.isFinite(value)
    ) ?? null;
}

function extractSleep(data) {
    const sleep = data?.dailySleepDTO ?? data;

    const seconds = numeric(
        sleep?.sleepTimeSeconds,
        sleep?.sleepDurationSeconds
    );

    return {
        hours: seconds === null ? null : seconds / 3600,

        score: numeric(
            sleep?.sleepScores?.overall?.value,
            sleep?.sleepScore,
            data?.sleepScores?.overall?.value
        )
    };
}

function extractBodyBattery(data, summary) {
    const days = Array.isArray(data) ? data : [data];

    const readings = days.flatMap(day =>
        Array.isArray(day?.bodyBatteryValuesArray)
            ? day.bodyBatteryValuesArray
            : []
    );

    const valid = readings
        .filter(reading =>
            Array.isArray(reading) &&
            typeof reading[0] === "number" &&
            typeof reading[1] === "number" &&
            Number.isFinite(reading[1]) &&
            reading[1] >= 0 &&
            reading[1] <= 100
        )
        .sort((a, b) => a[0] - b[0]);

    return numeric(
        valid.at(-1)?.[1],
        summary?.bodyBatteryMostRecentValue
    );
}

function extractHeartRate(data, summary) {
    return {
        resting: numeric(
            data?.restingHeartRate,
            summary?.restingHeartRate
        ),

        baseline: numeric(
            data?.lastSevenDaysAvgRestingHeartRate
        )
    };
}

function extractHRV(data) {
    const summary = data?.hrvSummary ?? data;

    return {
        value: numeric(summary?.lastNightAvg),

        baseline: numeric(summary?.weeklyAvg),

        status: typeof summary?.status === "string"
            ? summary.status
            : null
    };
}

function normalizeSummary(data, date) {
    if (Array.isArray(data)) {
        return data.find(item =>
            item?.calendarDate === date
        ) ?? null;
    }

    return data;
}

function normalizeActivities(data) {
    const activities = Array.isArray(data)
        ? data
        : data?.activities;

    if (!Array.isArray(activities)) return [];

    return activities.filter(activity =>
        activity && typeof activity === "object"
    );
}

function available(value) {
    return typeof value === "number" &&
        Number.isFinite(value);
}

function displayValue(value, decimals = 0) {
    return available(value)
        ? formatNumber(value, decimals)
        : "N/D";
}

function durationLabel(seconds) {
    if (!available(seconds) || seconds < 0) {
        return "N/D";
    }

    const minutes = Math.round(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const remainder = minutes % 60;

    return hours
        ? `${hours} h ${String(remainder).padStart(2, "0")}`
        : `${remainder} min`;
}

function activityType(activity) {
    const types = {
        running: "Course à pied",
        track_running: "Course sur piste",
        treadmill_running: "Course sur tapis",
        strength_training: "Musculation",
        indoor_cardio: "Cardio",
        cycling: "Cyclisme",
        walking: "Marche",
        swimming: "Natation",
        lap_swimming: "Natation en piscine",
        other: "Autre activité"
    };

    const type = activity.activityType?.typeKey;

    return types[type] || "Activité sportive";
}

function renderActivities(activities) {
    if (!activities.length) {
        return empty(
            "Aucune activité disponible",
            "Garmin n'a retourné aucune activité exploitable."
        );
    }

    return `
        <div class="table-container">
            <table class="data-table">

                <thead>
                    <tr>
                        <th>Activité</th>
                        <th>Date</th>
                        <th>Durée</th>
                        <th>Distance</th>
                        <th>FC moyenne</th>
                    </tr>
                </thead>

                <tbody>

                    ${activities.map(activity => {
                        const name =
                            activity.activityName ||
                            activityType(activity);

                        const date =
                            activity.startTimeLocal ||
                            activity.startTimeGMT ||
                            "";

                        const distance = numeric(
                            activity.distance
                        );

                        const heartRate = numeric(
                            activity.averageHR
                        );

                        return `
                            <tr>
                                <td>
                                    <strong>
                                        ${escapeHTML(name)}
                                    </strong>

                                    <p class="text-small text-muted">
                                        ${escapeHTML(activityType(activity))}
                                    </p>
                                </td>

                                <td>
                                    ${escapeHTML(
                                        String(date)
                                            .slice(0, 16)
                                            .replace("T", " ")
                                        || "N/D"
                                    )}
                                </td>

                                <td>
                                    ${durationLabel(
                                        numeric(activity.duration)
                                    )}
                                </td>

                                <td>
                                    ${distance === null
                                        ? "N/D"
                                        : `${formatNumber(distance / 1000, 2)} km`}
                                </td>

                                <td>
                                    ${heartRate === null
                                        ? "N/D"
                                        : `${formatNumber(heartRate, 0)} bpm`}
                                </td>
                            </tr>
                        `;
                    }).join("")}

                </tbody>

            </table>
        </div>
    `;
}

function providerStatus(label, result) {
    const success = result.status === "fulfilled";
    const hasData = success && result.value !== null;

    return `
        <div class="flex-row space-between wrap">
            <span>${escapeHTML(label)}</span>

            <span class="badge ${
                hasData
                    ? "badge-success"
                    : "badge-warning"
            }">
                ${hasData
                    ? "Réponse reçue"
                    : success
                        ? "Aucune donnée"
                        : "Indisponible"}
            </span>
        </div>
    `;
}

export function renderDataHub(container) {
    let requestVersion = 0;
    let importedIndicators = null;

    container.innerHTML = `
        <div class="grid grid-2">

            <article class="card stack">

                <header class="card-header">
                    <div>
                        <h2 class="card-title">Garmin Connect</h2>

                        <p class="card-description">
                            Données sportives personnelles.
                        </p>
                    </div>

                    <span class="metric-icon">
                        ${icon("activity")}
                    </span>
                </header>

                <div id="garmin-connection">
                    ${loading("Vérification de la configuration...")}
                </div>

                <form id="garmin-key-form" class="stack" method="post">
                    <div class="form-group">
                        <label for="garmin-key">Clé API personnelle Garmin</label>
                        <input id="garmin-key" type="password" autocomplete="off"
                               spellcheck="false" placeholder="GCDC_…" maxlength="1024" required>
                        <p class="form-help">La clé reste en mémoire jusqu’à la déconnexion ou au redémarrage du serveur.</p>
                    </div>
                    <div class="flex-row wrap">
                        <button id="garmin-key-save" class="button button-primary" type="submit">
                            Connecter et synchroniser
                        </button>
                        <button id="garmin-key-remove" class="button button-secondary" type="button" hidden>
                            Retirer la clé
                        </button>
                    </div>
                </form>

                <div class="form-group">
                    <label for="garmin-date">
                        Date des indicateurs
                    </label>

                    <input type="date"
                           id="garmin-date"
                           value="${localDate()}"
                           max="${localDate()}"
                           required/>
                </div>

                <button id="garmin-sync"
                        class="button button-primary button-block"
                        disabled>

                    ${icon("refresh")}
                    Synchroniser Garmin

                </button>

                <p id="garmin-sync-time"
                   class="text-small text-muted">
                    Aucune synchronisation effectuée.
                </p>

            </article>

            <article class="card stack">

                <header class="card-header">
                    <div>
                        <h2 class="card-title">
                            Sources environnementales
                        </h2>

                        <p class="card-description">
                            Services utilisés par TrackLab.
                        </p>
                    </div>

                    <span class="metric-icon">
                        ${icon("database")}
                    </span>
                </header>

                <div class="flex-row space-between">
                    <span>Open-Meteo Forecast</span>
                    <span class="badge badge-primary">Météo</span>
                </div>

                <div class="flex-row space-between">
                    <span>Open-Meteo Air Quality</span>
                    <span class="badge badge-primary">Atmosphère</span>
                </div>

                <div class="flex-row space-between">
                    <span>Open-Meteo Geocoding</span>
                    <span class="badge badge-primary">Recherche</span>
                </div>

                <div class="flex-row space-between">
                    <span>API Découpage administratif</span>
                    <span class="badge badge-primary">Communes</span>
                </div>

                <p class="text-small text-muted">
                    Les données environnementales sont mises en cache
                    côté serveur pour limiter les appels externes.
                </p>

            </article>

        </div>

        <h2 class="section-title">Indicateurs personnels</h2>

        <div id="garmin-metrics" class="grid grid-4">

            ${metric("Sommeil", "N/D", "h", "clock")}
            ${metric("Score de sommeil", "N/D", "/100", "activity")}
            ${metric("Body Battery", "N/D", "/100", "battery")}
            ${metric("FC au repos", "N/D", "bpm", "heart")}
            ${metric("VFC nocturne", "N/D", "ms", "activity")}
            ${metric("VFC moyenne sur 7 jours", "N/D", "ms", "activity")}
            ${metric("Nombre de pas", "N/D", "", "target")}
            ${metric("Calories totales", "N/D", "kcal", "activity")}

        </div>

        <article class="card stack">

            <h2 class="card-title">
                Exploiter mes données
            </h2>

            <p class="card-description">
                Transférez les indicateurs disponibles vers Training Lab
                pour les intégrer au TrackLab Score.
            </p>

            <div id="garmin-import-info" class="text-muted">
                Synchronisez Garmin pour récupérer vos indicateurs.
            </div>

            <button id="garmin-import"
                    class="button button-primary"
                    disabled>

                ${icon("activity")}
                Utiliser ces données dans Training Lab

            </button>

        </article>

        <article class="card stack">

            <header class="card-header">
                <div>
                    <h2 class="card-title">Activités récentes</h2>

                    <p class="card-description">
                        Les vingt dernières activités Garmin disponibles.
                    </p>
                </div>
            </header>

            <div id="garmin-activities">
                ${empty(
                    "En attente de synchronisation",
                    "Connectez Garmin pour consulter vos activités."
                )}
            </div>

        </article>

        <article class="card stack">

            <h2 class="card-title">État de la synchronisation</h2>

            <div id="garmin-endpoints" class="stack">
                <p class="text-muted">
                    Aucune requête personnelle effectuée.
                </p>
            </div>

        </article>

        <article class="card stack">

            <h2 class="card-title">
                Confidentialité et sécurité
            </h2>

            <p>
                La clé API Garmin est conservée exclusivement
                côté serveur dans le fichier de configuration privé.
            </p>

            <p>
                Les réponses Garmin sont mises en cache dans
                un répertoire personnel exclu du dépôt Git.
            </p>

            <p>
                Les données Garmin affichées ne sont pas enregistrées
                dans le stockage persistant du navigateur.
            </p>

            <p class="text-small text-muted">
                Une synchronisation peut utiliser une réponse
                encore valide du cache serveur.
                Les indicateurs ne remplacent pas une évaluation médicale.
            </p>

        </article>
    `;

    const connection = container.querySelector("#garmin-connection");
    const syncButton = container.querySelector("#garmin-sync");
    const dateInput = container.querySelector("#garmin-date");
    const metrics = container.querySelector("#garmin-metrics");
    const activities = container.querySelector("#garmin-activities");
    const endpoints = container.querySelector("#garmin-endpoints");
    const importButton = container.querySelector("#garmin-import");
    const importInfo = container.querySelector("#garmin-import-info");
    const syncTime = container.querySelector("#garmin-sync-time");
    const keyForm = container.querySelector("#garmin-key-form");
    const keyInput = container.querySelector("#garmin-key");
    const keySave = container.querySelector("#garmin-key-save");
    const keyRemove = container.querySelector("#garmin-key-remove");

    async function checkConnection() {
        const version = requestVersion;
        try {
            const status = await API.garminStatus();

            if (!connection.isConnected || version !== requestVersion) return;
            keyRemove.hidden = !status.configured;
            syncButton.disabled = !status.configured;

            if (status.configured) {
                connection.innerHTML = `
                    <div class="notice">
                        <strong class="text-success">
                            Clé API configurée
                        </strong>

                        <p>
                            Lancez une synchronisation pour vérifier
                            l'accès aux données Garmin.
                        </p>
                    </div>
                `;

                syncButton.disabled = false;

            } else {
                connection.innerHTML = `
                    <div class="notice notice-warning">
                        Garmin n'est pas configuré.
                        Ajoutez votre clé API personnelle ci-dessus.
                    </div>
                `;
            }

        } catch (error) {
            connection.innerHTML = errorState(error.message);
        }
    }

    async function synchronize() {
        const version = ++requestVersion;
        const date = dateInput.value;

        if (!date || date > localDate()) {
            toast("Sélectionnez une date valide.", "error");
            return;
        }

        syncButton.disabled = true;
        importButton.disabled = true;
        importedIndicators = null;

        metrics.innerHTML = loading(
            "Récupération des indicateurs Garmin..."
        );

        activities.innerHTML = loading(
            "Récupération des activités..."
        );

        endpoints.innerHTML = loading(
            "Synchronisation en cours..."
        );

        try {
            const requests = await Promise.allSettled([
                API.garminSleep(date),
                API.garminBodyBattery(date),
                API.garminHeartRate(date),
                API.garminHRV(date),
                API.garminSummary(date),
                API.garminActivities(20)
            ]);

            if (
                version !== requestVersion ||
                !metrics.isConnected
            ) return;

            const value = index =>
                requests[index].status === "fulfilled"
                    ? requests[index].value
                    : null;

            const summary = normalizeSummary(value(4), date);

            const sleep = extractSleep(value(0));
            const battery = extractBodyBattery(value(1), summary);
            const heart = extractHeartRate(value(2), summary);
            const hrv = extractHRV(value(3));

            const steps = numeric(summary?.totalSteps);
            const calories = numeric(summary?.totalKilocalories);

            const personalIndicators = {
                sleep_hours: sleep.hours,
                sleep_score: sleep.score,
                body_battery: battery,
                resting_heart_rate: heart.resting,
                baseline_heart_rate: heart.baseline,
                hrv: hrv.value,
                baseline_hrv: hrv.baseline
            };

            importedIndicators = Object.fromEntries(
                Object.entries(personalIndicators)
                    .filter(([, value]) => available(value))
            );

            const count = Object.keys(importedIndicators).length;

            metrics.innerHTML = `

                ${metric(
                    "Sommeil",
                    displayValue(sleep.hours, 1),
                    "h",
                    "clock"
                )}

                ${metric(
                    "Score de sommeil",
                    displayValue(sleep.score),
                    "/100",
                    "activity"
                )}

                ${metric(
                    "Body Battery",
                    displayValue(battery),
                    "/100",
                    "battery",
                    "Dernière valeur disponible"
                )}

                ${metric(
                    "FC au repos",
                    displayValue(heart.resting),
                    "bpm",
                    "heart"
                )}

                ${metric(
                    "VFC nocturne",
                    displayValue(hrv.value),
                    "ms",
                    "activity"
                )}

                ${metric(
                    "VFC moyenne sur 7 jours",
                    displayValue(hrv.baseline),
                    "ms",
                    "activity"
                )}

                ${metric(
                    "Nombre de pas",
                    displayValue(steps),
                    "",
                    "target"
                )}

                ${metric(
                    "Calories totales",
                    displayValue(calories),
                    "kcal",
                    "activity"
                )}

            `;

            activities.innerHTML = renderActivities(
                normalizeActivities(value(5))
            );

            const labels = [
                "Sommeil",
                "Body Battery",
                "Fréquence cardiaque",
                "Variabilité cardiaque",
                "Résumé quotidien",
                "Activités récentes"
            ];

            endpoints.innerHTML = requests.map(
                (result, index) =>
                    providerStatus(labels[index], result)
            ).join("");

            const successes = requests.filter(result =>
                result.status === "fulfilled"
            ).length;

            connection.innerHTML = `
                <div class="notice ${
                    successes < requests.length
                        ? "notice-warning"
                        : ""
                }">

                    <strong>
                        ${successes === requests.length
                            ? "Synchronisation réussie"
                            : successes > 0
                                ? "Synchronisation partielle"
                                : "Synchronisation indisponible"}
                    </strong>

                    <p>
                        ${successes}/${requests.length}
                        services ont répondu.
                    </p>

                </div>
            `;

            syncTime.textContent =
                `Dernière tentative : ${
                    new Date().toLocaleString("fr-FR")
                }`;

            importInfo.textContent = count
                ? `${count} champ(s) personnel(s) exploitable(s) pour le ${date}.`
                : "Aucun indicateur personnel exploitable.";

            importButton.disabled =
                count === 0 || date !== localDate();

            if (count && date !== localDate()) {
                importInfo.textContent +=
                    " Seules les données du jour peuvent alimenter le score actuel.";
            }

            if (successes > 0) {
                toast("Synchronisation Garmin terminée.");
            } else {
                toast("Garmin est actuellement indisponible.", "error");
            }

        } catch (error) {
            metrics.innerHTML = errorState(error.message);
            activities.innerHTML = errorState(error.message);
            endpoints.innerHTML = errorState(error.message);

        } finally {
            if (version === requestVersion) {
                syncButton.disabled = false;
            }
        }
    }

    syncButton.addEventListener("click", synchronize);

    keyForm.addEventListener("submit", async event => {
        event.preventDefault();
        if (keySave.disabled) return;
        const key = keyInput.value.trim();
        if (!key) return;
        keySave.disabled = keyRemove.disabled = true;
        try {
            await API.setGarminKey(key);
            keyForm.reset();
            keyRemove.hidden = false;
            if (keyForm.isConnected) await synchronize();
        } catch (error) {
            toast(error.message, "error");
        } finally {
            keySave.disabled = keyRemove.disabled = false;
        }
    });

    keyRemove.addEventListener("click", async () => {
        keySave.disabled = keyRemove.disabled = true;
        try {
            await API.setGarminKey("");
            requestVersion++;
            if (keyForm.isConnected) renderDataHub(container);
            toast("Clé Garmin retirée.", "success");
        } catch (error) {
            toast(error.message, "error");
            keySave.disabled = keyRemove.disabled = false;
        }
    });

    dateInput.addEventListener("change", () => {
        requestVersion++;
        importedIndicators = null;
        importButton.disabled = true;
        syncButton.disabled = false;

        importInfo.textContent =
            "Synchronisez la nouvelle date avant tout transfert.";
    });

    importButton.addEventListener("click", async () => {
        if (
            !importedIndicators ||
            dateInput.value !== localDate()
        ) return;

        importButton.disabled = true;

        try {
            const current = Store.get("training");

            const merged = {
                ...current,
                ...importedIndicators
            };

            await API.analyzeTraining(merged);

            await Store.set("training", merged);
            Store.set("analysis", null);

            toast("Indicateurs Garmin transférés.");

            location.hash = "training";

        } catch (error) {
            toast(error.message, "error");
            importButton.disabled = false;
        }
    });

    checkConnection();
}
