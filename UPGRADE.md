# Mettre à jour son instance CampMenu

CampMenu compile son binaire depuis le code source à chaque démarrage du pod :
mettre à jour revient à changer la référence git pointée par `REPO_REF`
(ConfigMap `campmenu-config`) puis à redémarrer les déploiements. **Regardez
toujours le `CHANGELOG.md` de la version cible avant de mettre à jour** :
toute entrée modifiant la base de données est signalée par un bloc
`### ⚠️ Base de données`.

## Cas 1 — Sans changement de base de données

1. Changez `REPO_REF` dans le ConfigMap `campmenu-config`.
2. Relancez les déploiements :
   ```
   kubectl -n <namespace> rollout restart deployment/campmenu-backend deployment/campmenu-frontend
   ```

## Cas 2 — Avec changement de base de données

1. Mettez d'abord à jour le code (Cas 1 ci-dessus) — le nouveau backend sait
   lire l'ancien schéma le temps de la migration.
2. Allez dans **Admin → Mise à jour**.
3. Le fichier de migration SQL correspondant à cette version est proposé
   automatiquement. Lancez d'abord **Tester (dry-run)** — la requête est
   exécutée puis annulée, sans toucher vos données, pour vérifier qu'elle
   s'applique proprement à votre base.
4. Si le test réussit, cliquez **Appliquer** pour l'exécuter réellement.

Les migrations s'appliquent une par une et dans l'ordre : impossible de sauter
une version ou d'en appliquer une hors séquence. La version de schéma actuelle
et la prochaine migration en attente sont toujours visibles dans cet écran.
