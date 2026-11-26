/**
 * TrackLab - Profil sportif
 * Gestion de l'identité sportive et de la localisation.
 */

import { API } from "../api.js";

import {
    Store,
    EVENTS,
    OBJECTIVES,
    formatNumber
} from "../store.js";

import {
    escapeHTML,
    selectOptions,
    icon,
    toast
} from "../ui.js";

function cityLabel(city) {
    if (!city) {
        return "Aucune ville sélectionnée";
    }

    return [
        city.name,
        city.department,
        city.country
    ].filter(Boolean).join(" · ");
}

export function renderProfile(container) {
    const profile = Store.get("profile");

    let selectedCity = profile.city;
    let searchVersion = 0;

    container.innerHTML = `
        <form id="profile-form" class="stack">

            <div class="grid grid-2">

                <article class="card">

                    <header class="card-header">
                        <div>
                            <h2 class="card-title">Identité sportive</h2>

                            <p class="card-description">
                                Adaptez TrackLab à votre pratique.
                            </p>
                        </div>

                        <span class="metric-icon">
                            ${icon("user")}
                        </span>
                    </header>

                    <div class="stack">

                        <div class="form-group">

                            <label for="profile-name">
                                Nom ou prénom
                            </label>

                            <input
                                id="profile-name"
                                maxlength="60"
                                autocomplete="given-name"
                                placeholder="Votre prénom"
                                value="${escapeHTML(profile.name)}"
                            >

                            <span class="form-help">
                                Facultatif, conservé uniquement
                                dans votre navigateur.
                            </span>

                        </div>

                        <div class="form-group">

                            <label for="profile-event">
                                Épreuve principale
                            </label>

                            <select id="profile-event">
                                ${selectOptions(EVENTS, profile.event)}
                            </select>

                        </div>

                        <div class="form-group">

                            <label for="profile-objective">
                                Objectif d'entraînement
                            </label>

                            <select id="profile-objective">
                                ${selectOptions(
                                    OBJECTIVES,
                                    profile.objective
                                )}
                            </select>

                        </div>

                    </div>

                </article>

                <article class="card">

                    <header class="card-header">
                        <div>
                            <h2 class="card-title">Localisation</h2>

                            <p class="card-description">
                                Définissez votre lieu habituel
                                d'entraînement.
                            </p>
                        </div>

                        <span class="metric-icon">
                            ${icon("location")}
                        </span>
                    </header>

                    <div class="stack">

                        <div class="notice">
                            <div>
                                <strong>Ville sélectionnée</strong>

                                <p id="selected-city">
                                    ${escapeHTML(cityLabel(selectedCity))}
                                </p>
                            </div>
                        </div>

                        <div class="form-group">

                            <label for="city-search">
                                Rechercher une ville
                            </label>

                            <div class="flex-row">

                                <input
                                    id="city-search"
                                    placeholder="Ex. Bordeaux, Paris, Londres"
                                    maxlength="100"
                                    autocomplete="off"
                                >

                                <button
                                    class="button button-secondary"
                                    id="city-search-button"
                                    type="button"
                                >
                                    Rechercher
                                </button>

                            </div>

                        </div>

                        <button
                            type="button"
                            id="gps-button"
                            class="button button-secondary button-block"
                        >
                            ${icon("location")}
                            Utiliser ma position
                        </button>

                        <p class="form-help">
                            La géolocalisation propose les communes
                            françaises situées à proximité.
                            Votre position GPS brute n'est pas conservée.
                        </p>

                        <div
                            id="city-results"
                            class="search-results"
                            aria-live="polite"
                        ></div>

                    </div>

                </article>

            </div>

            <div class="form-actions">

                <button
                    class="button button-primary"
                    type="submit"
                >
                    ${icon("check")}
                    Enregistrer mon profil
                </button>

                <a
                    href="#dashboard"
                    class="button button-secondary"
                >
                    Retour au tableau de bord
                </a>

            </div>

        </form>

        <div class="notice">
            <div>
                <strong>Confidentialité</strong>

                <p>
                    Votre profil est enregistré localement
                    dans votre navigateur.
                    Les coordonnées de la ville sélectionnée
                    sont utilisées pour interroger les services
                    environnementaux.
                </p>
            </div>
        </div>
    `;

    const find = id => container.querySelector(`#${id}`);

    const form = find("profile-form");
    const results = find("city-results");
    const searchInput = find("city-search");
    const searchButton = find("city-search-button");
    const gpsButton = find("gps-button");

    function selectCity(city) {
        selectedCity = city;

        find("selected-city").textContent = cityLabel(city);

        results.replaceChildren();

        toast(
            `${city.name} sélectionnée. Pensez à enregistrer votre profil.`,
            "success"
        );
    }

    function displayCities(cities) {
        results.replaceChildren();

        if (!cities.length) {
            results.textContent = "Aucune ville trouvée.";
            return;
        }

        for (const city of cities) {
            const button = document.createElement("button");

            button.type = "button";
            button.className = "search-result";

            const information = document.createElement("div");
            const name = document.createElement("strong");
            const details = document.createElement("small");

            name.textContent = city.name;

            details.textContent = [
                city.department,
                city.country
            ].filter(Boolean).join(" · ");

            information.className = "stack";

            information.append(name, details);
            button.append(information);

            if (Number.isFinite(city.distance)) {
                const distance = document.createElement("small");

                distance.textContent =
                    `${formatNumber(city.distance)} km`;

                button.append(distance);
            }

            button.addEventListener("click", () => {
                selectCity(city);
            });

            results.append(button);
        }
    }

    async function searchCities() {
        const query = searchInput.value.trim();

        if (query.length < 2) {
            toast(
                "Saisissez au moins deux caractères.",
                "error"
            );

            return;
        }

        const version = ++searchVersion;

        searchButton.disabled = true;
        results.textContent = "Recherche en cours...";

        try {
            const response = await API.searchCities(query);

            if (
                version !== searchVersion ||
                !results.isConnected
            ) return;

            const cities = (response.results || [])
                .filter(city =>
                    typeof city.name === "string" &&
                    Number.isFinite(city.latitude) &&
                    Number.isFinite(city.longitude)
                )
                .map(city => ({
                    id: city.id,
                    name: city.name,
                    country: city.country,
                    department: city.admin1 || "",
                    latitude: city.latitude,
                    longitude: city.longitude
                }));

            displayCities(cities);

        } catch (error) {
            if (version === searchVersion) {
                results.textContent = error.message;
            }

        } finally {
            searchButton.disabled = false;
        }
    }

    searchButton.addEventListener("click", searchCities);

    searchInput.addEventListener("keydown", event => {
        if (event.key === "Enter") {
            event.preventDefault();
            searchCities();
        }
    });

    gpsButton.addEventListener("click", async () => {
        const version = ++searchVersion;

        gpsButton.disabled = true;

        results.textContent =
            "Recherche des communes à proximité...";

        try {
            if (typeof TrackLabGeolocation === "undefined") {
                throw new Error(
                    "Service de géolocalisation indisponible."
                );
            }

            const cities =
                await TrackLabGeolocation.getNearbyCities();

            if (
                version === searchVersion &&
                results.isConnected
            ) {
                displayCities(cities);
            }

        } catch {
            if (version === searchVersion) {
                results.textContent =
                    "Position indisponible. Autorisez la géolocalisation ou recherchez votre ville manuellement.";
            }

        } finally {
            gpsButton.disabled = false;
        }
    });

    form.addEventListener("submit", async event => {
        event.preventDefault();

        if (!selectedCity) {
            toast(
                "Sélectionnez une ville avant d'enregistrer.",
                "error"
            );

            return;
        }

        try {
            const previousProfile = Store.get("profile");
            const previousCity = previousProfile.city;

            await Store.updateProfile({
                name: find("profile-name").value.trim(),
                event: find("profile-event").value,
                objective: find("profile-objective").value,
                trackDirection: previousProfile.trackDirection,
                city: selectedCity
            });

            if (
                previousCity?.latitude !== selectedCity.latitude ||
                previousCity?.longitude !== selectedCity.longitude
            ) {
                Store.set("environment", null);
            }

            Store.set("analysis", null);

            toast("Profil sportif enregistré.", "success");

            location.hash = "dashboard";

        } catch (error) {
            toast(error.message, "error");
        }
    });
}
