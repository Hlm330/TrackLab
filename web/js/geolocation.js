"use strict";

const TrackLabGeolocation = (() => {
    const EARTH_RADIUS_KM = 6371;
    const MAX_NEARBY_DISTANCE_KM = 100;

    let communesPromise = null;

    function toRadians(degrees) {
        return degrees * Math.PI / 180;
    }

    function validCoordinates(latitude, longitude) {
        return Number.isFinite(latitude) &&
            Number.isFinite(longitude) &&
            latitude >= -90 &&
            latitude <= 90 &&
            longitude >= -180 &&
            longitude <= 180;
    }

    // Calcule la distance à vol d'oiseau entre deux positions.
    function distanceKm(lat1, lon1, lat2, lon2) {
        if (!validCoordinates(lat1, lon1) ||
            !validCoordinates(lat2, lon2)) {
            throw new Error("Coordonnées géographiques invalides.");
        }

        const deltaLat = toRadians(lat2 - lat1);
        const deltaLon = toRadians(lon2 - lon1);

        const a =
            Math.sin(deltaLat / 2) ** 2 +
            Math.cos(toRadians(lat1)) *
            Math.cos(toRadians(lat2)) *
            Math.sin(deltaLon / 2) ** 2;

        return 2 * EARTH_RADIUS_KM *
            Math.asin(Math.sqrt(Math.min(1, Math.max(0, a))));
    }

    // Charge la base locale une seule fois par page.
    async function loadCommunes() {
        if (!communesPromise) {
            communesPromise = fetch("/assets/communes.json")
                .then(response => {
                    if (!response.ok) {
                        throw new Error(
                            "Base des communes indisponible."
                        );
                    }

                    return response.json();
                })
                .then(data => {
                    if (!Array.isArray(data)) {
                        throw new Error(
                            "Base géographique invalide."
                        );
                    }

                    return data
                        .filter(commune => {
                            const coords = commune.centre?.coordinates;

                            return typeof commune.nom === "string" &&
                                typeof commune.code === "string" &&
                                Array.isArray(coords) &&
                                validCoordinates(coords[1], coords[0]);
                        })
                        .map(commune => ({
                            id: commune.code,
                            name: commune.nom,
                            department: commune.codeDepartement || "",
                            latitude: commune.centre.coordinates[1],
                            longitude: commune.centre.coordinates[0],
                            country: "France"
                        }));
                })
                .catch(error => {
                    communesPromise = null;
                    throw error;
                });
        }

        return communesPromise;
    }

    // Recherche les communes les plus proches.
    async function nearestCities(latitude, longitude) {
        if (!validCoordinates(latitude, longitude)) {
            throw new Error("Position géographique invalide.");
        }

        const communes = await loadCommunes();

        return communes
            .map(commune => ({
                ...commune,
                distance: distanceKm(
                    latitude,
                    longitude,
                    commune.latitude,
                    commune.longitude
                )
            }))
            .filter(commune =>
                commune.distance <= MAX_NEARBY_DISTANCE_KM
            )
            .sort((a, b) =>
                a.distance - b.distance ||
                a.id.localeCompare(b.id)
            )
            .slice(0, 10);
    }

    // Demande explicitement l'autorisation de localisation.
    function getCurrentPosition() {
        return new Promise((resolve, reject) => {
            if (!navigator.geolocation) {
                reject(new Error(
                    "La géolocalisation est indisponible."
                ));
                return;
            }

            navigator.geolocation.getCurrentPosition(
                position => {
                    resolve({
                        latitude: position.coords.latitude,
                        longitude: position.coords.longitude,
                        accuracy: position.coords.accuracy
                    });
                },
                error => {
                    const messages = {
                        1: "Autorisation de localisation refusée.",
                        2: "Position géographique indisponible.",
                        3: "Délai de localisation dépassé."
                    };

                    reject(new Error(
                        messages[error.code] ||
                        "Impossible de déterminer votre position."
                    ));
                },
                {
                    enableHighAccuracy: false,
                    timeout: 10000,
                    maximumAge: 60000
                }
            );
        });
    }

    async function getNearbyCities() {
        const position = await getCurrentPosition();

        return nearestCities(
            position.latitude,
            position.longitude
        );
    }

    return Object.freeze({
        distanceKm,
        nearestCities,
        getNearbyCities
    });
})();
