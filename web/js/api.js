/**
 * TrackLab - Client API
 * Centralise les communications avec le serveur Go.
 */

import { accountSessionHeaders } from "./store.js";

async function request(path, options = {}) {
    const controller = new AbortController();

    const timeout = setTimeout(() => {
        controller.abort();
    }, 45000);

    try {
        const response = await fetch(path, {
            ...options,
            signal: controller.signal,
            credentials: "same-origin",
            cache: "no-store",
            headers: {
                Accept: "application/json",
                ...(path.startsWith("/api/garmin/") ? accountSessionHeaders() : {}),
                ...options.headers
            }
        });

        const data = await response.json();

        if (!response.ok) {
            throw new Error(
                data.error ||
                data.message ||
                `Erreur du serveur (${response.status}).`
            );
        }

        return data;

    } catch (error) {
        if (error.name === "AbortError") {
            throw new Error("Le serveur met trop de temps à répondre.");
        }

        if (error instanceof TypeError) {
            throw new Error("Impossible de joindre le serveur TrackLab.");
        }

        if (error instanceof SyntaxError) {
            throw new Error("Réponse du serveur invalide.");
        }

        throw error;

    } finally {
        clearTimeout(timeout);
    }
}

function post(path, data) {
    return request(path, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify(data)
    });
}

function coordinates(latitude, longitude) {
    return new URLSearchParams({
        lat: latitude,
        lon: longitude
    }).toString();
}

export const API = Object.freeze({

    health() {
        return request("/api/health");
    },

    searchCities(name) {
        const params = new URLSearchParams({ q: name });

        return request(`/api/cities?${params}`);
    },

    weather(latitude, longitude) {
        return request(
            `/api/weather?${coordinates(latitude, longitude)}`
        );
    },

    airQuality(latitude, longitude) {
        return request(
            `/api/air-quality?${coordinates(latitude, longitude)}`
        );
    },

    async environment(latitude, longitude) {
        const [weather, airQuality] = await Promise.all([
            this.weather(latitude, longitude),
            this.airQuality(latitude, longitude)
        ]);

        return {
            weather,
            airQuality,
            fetchedAt: Date.now()
        };
    },

    analyzeSession(data) {
        return post("/api/analysis/session", data);
    },

    analyzeWind(data) {
        return post("/api/analysis/wind", data);
    },

    analyzeTraining(data) {
        return post("/api/analysis/training", data);
    },

    analyzeLoad(sessions, referenceDate) {
        return post("/api/analysis/load", {
            sessions,
            reference_date: referenceDate
        });
    },

    garminStatus() {
        return request("/api/garmin/status");
    },

    setGarminKey(key) {
        return request("/api/garmin/key", {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ key })
        });
    },

    garminActivities(limit = 20) {
        return request(
            `/api/garmin/activities?limit=${encodeURIComponent(limit)}`
        );
    },

    garminSleep(date) {
        return request(
            `/api/garmin/sleep?date=${encodeURIComponent(date)}`
        );
    },

    garminBodyBattery(date) {
        return request(
            `/api/garmin/body-battery?date=${encodeURIComponent(date)}`
        );
    },

    garminHeartRate(date) {
        return request(
            `/api/garmin/heart-rate?date=${encodeURIComponent(date)}`
        );
    },

    garminHRV(date) {
        return request(
            `/api/garmin/hrv?date=${encodeURIComponent(date)}`
        );
    },

    garminSummary(date) {
        return request(
            `/api/garmin/summary?date=${encodeURIComponent(date)}`
        );
    }

});
