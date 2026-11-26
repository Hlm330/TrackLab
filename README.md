# TrackLab

TrackLab aide les athlètes à préparer une séance d’athlétisme. Il croise la météo, le vent et les indicateurs de préparation pour éclairer une décision d’entraînement. Le score est un repère heuristique sur 100.

## Réponse à la consigne

- **Outil du quotidien et visualisation.** TrackLab aide à planifier une séance d’athlétisme. Le tableau de bord, les jauges, la représentation de la piste et les graphiques rendent les données lisibles.
- **Étude de scénarios.** Dans Session Planner, comparer deux horaires pour une même discipline. Dans WindLab, modifier la direction de course pour observer comment un même vent devient plus favorable ou plus pénalisant.
- **API.** Open-Meteo fournit la météo et la qualité de l’air. Garmin est une source optionnelle pour les indicateurs personnels. Le projet peut être démontré sans clé Garmin.
- **Code sur Git.** [Dépôt TrackLab](https://github.com/Hlm330/TrackLab).

### Règles d’analyse implémentées

| Règle | Condition et résultat |
| --- | --- |
| Chaleur ressentie | Dès 35 °C ressentis, un avertissement invite à privilégier un créneau plus frais pour les efforts intenses. |
| Chaleur humide | À partir de 25 °C et de 75 % d’humidité relative, le moteur météo applique une pénalité de 10 points et conseille d’adapter les récupérations. |
| Charge d’entraînement | La charge est calculée en multipliant la durée en minutes par l’effort perçu de 0 à 10. Une séance de 40 minutes à 6 vaut 240 unités arbitraires. |
| Vent arrière | Pour les épreuves concernées, une composante de vent arrière supérieure à 2,0 m/s déclenche un signalement réglementaire. |

### Utilisation de l’IA et garde-fous

ChatGPT a accompagné l’analyse du projet et Codex a aidé à inspecter le code, affiner les choix techniques, implémenter des modifications ciblées, écrire des tests et préparer la documentation. La méthode de prompt a consisté à donner un objectif concret, préciser le périmètre et les exclusions, demander une vérification, puis corriger à partir du résultat. Par exemple, la demande sur les animations imposait d’inspecter l’existant, de respecter la préférence de mouvement réduit et de préserver les moteurs et Chart.js. La session Garmin a été isolée après avoir identifié que le jeton global n’était rattaché à aucun compte.

Les règles appliquées pendant la collaboration ont été de préserver les secrets et les données personnelles, d’utiliser des comptes et clés fictifs pour les tests, et de signaler ce qui reste à vérifier. Ces précautions documentent les pratiques du projet. La charte éthique écrite est une pièce distincte à joindre si elle doit figurer dans le rendu.

**Lien de conversation pour le rendu :** ajouter le lien public produit par la fonction **Partager** de ChatGPT pour « Analyse projet informatique » et les autres échanges IA utilisés, après avoir vérifié qu’ils ne révèlent aucune donnée privée. L’identifiant de la conversation d’origine est privé et ne constitue pas un lien de partage.

## Lancement local

Prérequis : Go 1.27.1 et un navigateur récent. Depuis la racine du dépôt :

```sh
go run ./cmd/server
```

Ouvrir [http://127.0.0.1:8080](http://127.0.0.1:8080), puis choisir **Créer un compte**. Utiliser une adresse e-mail et un mot de passe d’au moins 12 caractères. Si une ancienne instance est déjà ouverte dans le terminal, l’arrêter avec Ctrl+C avant de relancer la commande. Le serveur écoute uniquement sur une adresse de boucle locale. Le réseau est nécessaire pour les données météo et Garmin.

## Parcours de démonstration

| Module | À montrer |
| --- | --- |
| Profil sportif | Choisir une discipline, un objectif, une commune et l’orientation de piste. |
| Tableau de bord | Consulter le score de séance et la synthèse des conditions. |
| WindLab | Comparer le vent de face, arrière et latéral selon les zones de la piste. |
| Session Planner | Comparer les créneaux et repérer la meilleure option proposée. |
| Training Lab | Renseigner la préparation, ajouter une séance, consulter l’historique et les charges à 7 et 28 jours. |
| Data Hub | Afficher les sources, synchroniser Garmin et transférer les indicateurs du jour vers Training Lab. |

Pour la démonstration, commencer par le profil, puis ouvrir le tableau de bord. Dans Training Lab, saisir une durée et un effort perçu pour enregistrer une séance. Se déconnecter puis se reconnecter pour vérifier qu’elle est toujours présente.

## Connecter Garmin

1. Ouvrir **Data Hub** et coller sa clé API personnelle Garmin, qui commence par `GCDC_`.
2. Cliquer sur **Connecter et synchroniser**. Le serveur échange la clé contre un jeton d’accès Garmin.
3. Consulter les 20 activités récentes disponibles et les indicateurs de la date choisie : sommeil, Body Battery, fréquence cardiaque, VFC et résumé quotidien. La couverture dépend des autorisations, de l’appareil et des données présentes chez Garmin.
4. Pour la date du jour, utiliser **Utiliser ces données dans Training Lab** pour enregistrer les indicateurs exploitables dans le compte.

La clé est masquée à la saisie et conservée en mémoire dans la session. Elle disparaît à la déconnexion, à l’expiration de la session ou au redémarrage. **Retirer la clé** permet de couper la connexion sur-le-champ. Il faut la saisir de nouveau pour une autre session. Les indicateurs explicitement transférés restent dans le compte. Une synchronisation peut être partielle et affiche le nombre de services ayant répondu.

## Architecture et données

- `cmd/server/` démarre le serveur HTTP local et sert les ressources publiques.
- `web/` contient le frontend HTML, CSS et JavaScript, les six modules et Chart.js fourni localement. Les animations tiennent compte de la préférence de mouvement réduit.
- `internal/handlers/` regroupe les routes HTTP, les comptes et l’intégration Garmin par session.
- `internal/analysis/` contient les moteurs de météo, de vent, de préparation et de charge. Ils tournent également sans Garmin avec les informations saisies manuellement.
- `internal/api/` appelle les services externes. Le cache disque des données environnementales se trouve dans `data/cache/`.

Le TrackLab Score combine météo, vent et préparation avec des pondérations liées à l’objectif. Une couverture personnelle faible réduit le poids de la préparation. La charge d’une séance est la durée en minutes multipliée par l’effort perçu. Les scores aident à comparer des conditions et ne constituent pas un avis médical.

Les comptes, profils, séances et indicateurs enregistrés sont stockés localement dans `data/personal/accounts.json`, exclu de Git. Les mots de passe sont hachés avec PBKDF2-SHA-256 et un sel individuel. Le navigateur reçoit un cookie de session HttpOnly. Les sessions sont invalidées au redémarrage du serveur. Le bouton **Importer les anciennes données** permet de copier volontairement le profil et les séances hérités du navigateur, sans remplacer les données déjà présentes dans le compte.

## Vérifications

```sh
go test ./...
git diff --check
```

Les tests couvrent les moteurs, le cache, la persistance des séances, la déconnexion et l’isolation entre comptes. Les scénarios Garmin utilisent une clé fictive pour tester son isolation et sa disparition à la déconnexion, sans requête Garmin réelle. Pour une validation manuelle, vérifier le parcours dans le navigateur, puis faire une synchronisation avec sa propre clé si elle est disponible.

## Périmètre

TrackLab est conçu pour une démonstration et un usage local. Les données météo et de qualité de l’air proviennent d’Open-Meteo. Les données Garmin dépendent de la disponibilité du service et des autorisations de la clé. La synchronisation porte sur les endpoints intégrés et les activités récentes, sans prétendre télécharger tout l’historique Garmin Connect. Ne pas joindre de fichier de comptes, de cache personnel ni de configuration privée au rendu.

