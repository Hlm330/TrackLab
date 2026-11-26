/**
 * TrackLab - Traitement des données environnementales
 */

function requireNumber(value, label) {
    if (typeof value !== "number" || !Number.isFinite(value)) {
        throw new Error(`Donnée indisponible : ${label}.`);
    }

    return value;
}

export function currentHour(environment) {
    const time = environment.weather?.current?.time;

    if (typeof time !== "string") {
        throw new Error("Horaire météorologique indisponible.");
    }

    return `${time.slice(0, 13)}:00`;
}

export function environmentAt(environment, time) {
    const weather = environment.weather?.hourly;
    const air = environment.airQuality?.hourly;

    const weatherIndex = weather?.time?.indexOf(time) ?? -1;
    const airIndex = air?.time?.indexOf(time) ?? -1;

    if (weatherIndex < 0 || airIndex < 0) {
        throw new Error("Données environnementales incomplètes pour ce créneau.");
    }

    const w = (key, label) =>
        requireNumber(weather[key]?.[weatherIndex], label);

    const a = (key, label) =>
        requireNumber(air[key]?.[airIndex], label);

    return {
        time,

        weather: {
            temperature: w("temperature_2m", "température"),
            apparent_temperature: w("apparent_temperature", "ressenti"),
            humidity: w("relative_humidity_2m", "humidité"),
            precipitation: w("precipitation", "précipitations"),
            precipitation_probability: w(
                "precipitation_probability",
                "probabilité de pluie"
            ),
            uv_index: w("uv_index", "indice UV"),
            wind_gusts: w("wind_gusts_10m", "rafales"),
            air_quality_index: a("european_aqi", "qualité de l'air")
        },

        wind: {
            speed: w("wind_speed_10m", "vitesse du vent"),
            direction: w("wind_direction_10m", "direction du vent")
        }
    };
}

export function sessionInput(profile, conditions, training = {}) {
    return {
        event: profile.event,
        objective: profile.objective,
        weather: conditions.weather,
        wind: {
            ...conditions.wind,
            track_direction: profile.trackDirection
        },
        training
    };
}

export function airQualityLabel(value) {
    if (value <= 20) return "Bonne";
    if (value <= 40) return "Acceptable";
    if (value <= 60) return "Modérée";
    if (value <= 80) return "Mauvaise";
    if (value <= 100) return "Très mauvaise";

    return "Extrêmement mauvaise";
}

export function cardinalDirection(angle) {
    const directions = [
        "Nord", "Nord-Est", "Est", "Sud-Est",
        "Sud", "Sud-Ouest", "Ouest", "Nord-Ouest"
    ];

    return directions[Math.round(angle / 45) % 8];
}

export function signedNumber(value, decimals = 1) {
    if (!Number.isFinite(value)) return "N/D";

    const formatted = new Intl.NumberFormat("fr-FR", {
        maximumFractionDigits: decimals
    }).format(value);

    return value > 0 ? `+${formatted}` : formatted;
}

export function formatHour(time) {
    return time.slice(11, 16);
}
