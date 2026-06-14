# Garde-fou d'identité HFR (issue #32)

**Date** : 2026-06-14
**Issue** : [#32](https://github.com/ForumHFR/hfr-mcp/issues/32) — écritures sous le mauvais compte HFR
**Statut** : design validé, prêt pour le plan d'implémentation

## Problème

Les outils d'écriture (`hfr_reply`, `hfr_edit`, `hfr_mp`, `hfr_create_topic`) publient sous le compte HFR
déterminé par la configuration de la session (`HFR_LOGIN` / `hfr.conf`). Ce compte peut dériver
silencieusement d'une session à l'autre, et **rien ne permet de connaître le compte actif avant d'écrire**.
Résultat constaté plusieurs fois : des messages destinés au compte agent sont partis sous un autre compte.

Le forum n'offre aucun outil de suppression : une écriture sous le mauvais compte doit être retirée à la main.
La prévention est donc la seule protection viable.

## Objectifs

1. **Savoir** quel compte est actif avant d'écrire (`hfr_whoami` / `hfr whoami`).
2. **Refuser** d'écrire quand le compte connecté ne correspond pas au compte voulu — déclaré soit côté serveur,
   soit par commande (défense en profondeur).
3. **Tracer** systématiquement, dans le retour de chaque écriture réussie, le compte effectivement utilisé.

Hors scope : les bugs de l'issue #31 (notif e-mail, cache `hfr_read`), un éventuel `hfr_delete`,
la sélection/bascule de compte au runtime.

## Décisions de design

| Décision | Choix retenu |
|---|---|
| Déclaration du compte attendu | **Défense en profondeur** : barrière serveur `HFR_EXPECT_LOGIN` (env/`hfr.conf`) **et** paramètre par appel (`--pseudo` CLI / `expect` MCP). |
| Critère de comparaison | **Pseudo + userId** : une contrainte matche si elle égale le pseudo (insensible à la casse) **ou** le userId numérique. |
| Comportement par défaut (aucun compte attendu) | **Fail-open** : l'écriture passe, mais le retour affiche toujours le compte utilisé. Le garde-fou strict ne s'active que lorsqu'un compte attendu est déclaré. |
| Emplacement de la garde | Dans le **`Client` HFR** (`internal/hfr`), point de passage unique pour la CLI et le MCP. |

## Architecture

La logique vit dans le `Client` HFR. La CLI et le serveur MCP ne font que :
- fournir le compte attendu issu de l'appel (`--pseudo` / champ `expect`) ;
- fournir le compte attendu serveur (`HFR_EXPECT_LOGIN`) au moment de la construction du client ;
- formater l'identité renvoyée par le client.

Aucune vérification n'est dupliquée dans les couches d'appel.

## Composants

### 1. `internal/hfr` — résolution d'identité

- `Client` gagne deux champs : `userID string` et `expectedLogin string`.
- `type Identity struct { Pseudo string; UserID string; Authenticated bool }`.
- Au login (dans/après `fetchHashCheck`, qui charge déjà `/user/editprofil.php`), résoudre le userId réel.
  Source par ordre de préférence : cookie `md_user_id` s'il existe, sinon parse de la page profil
  (lien/champ exposant l'identifiant). **Si le userId ne peut pas être résolu, `userID` reste vide** et la
  garde se rabat sur le pseudo seul — pas d'échec dur.
- `func (c *Client) Whoami() Identity` — renvoie l'identité en cache (zéro requête supplémentaire ;
  l'appelant déclenche le login lazy au préalable).

### 2. `internal/hfr` — garde-fou

```go
func (c *Client) checkIdentity(expect string) error {
    for _, want := range []string{c.expectedLogin, expect} {
        if want == "" {
            continue // fail-open : contrainte absente
        }
        if !c.identityMatches(want) {
            return &HfrError{Code: "identity", Message: ...}
        }
    }
    return nil
}

func (c *Client) identityMatches(want string) bool {
    return strings.EqualFold(want, c.pseudo) || (c.userID != "" && want == c.userID)
}
```

- Appelé en tête de `Reply`, `Edit`, `CreateTopic`, `SendMP`, **après** `ensureAuth`/login et **avant** tout POST.
- `expectedLogin` (serveur) et `expect` (appel) sont deux contraintes indépendantes : les deux doivent passer.

### 3. `internal/config` — compte attendu serveur

- Lire `HFR_EXPECT_LOGIN` (env) et `expect_login=` dans `hfr.conf`. Optionnel. L'env prime sur le fichier.
- Transmis au `Client` à la construction.

### 4. `internal/mcp` — exposition

- Nouvel outil **`hfr_whoami`** (sans paramètre) : déclenche le login lazy, renvoie pseudo + userId +
  compte attendu serveur.
- Champ optionnel `expect string` ajouté à `ReplyInput`, `EditInput`, `CreateTopicInput`, `MPInput`,
  transmis à la méthode du client.
- Le serveur passe `HFR_EXPECT_LOGIN` au client au démarrage.
- Les handlers d'écriture, en cas de succès, formatent le compte effectif via `client.Whoami()`.

### 5. `cmd/hfr` — CLI

- Flag global **`--pseudo <login>`** : sur les commandes d'écriture, transmis comme `expect`.
- Nouvelle sous-commande **`hfr whoami`** : login puis affichage pseudo + userId + compte attendu.
- Lit aussi `HFR_EXPECT_LOGIN` via la config.

## Comportement (flux)

**Écriture** (`reply`/`edit`/`mp`/`new`) :
1. login lazy → résolution de l'identité (pseudo + userId) ;
2. `checkIdentity(expect)` contre `HFR_EXPECT_LOGIN` et le paramètre d'appel ;
3. si une contrainte échoue → erreur `identity`, **aucun POST** ;
4. sinon POST, puis retour incluant le compte effectif.

**`whoami`** : login → renvoie l'identité. Lecture seule, ne poste rien.

## Formats de message

- Succès d'écriture : `Message posté sous xatelitte (userId 1214571).`
- Refus : `écriture refusée : compte connecté "XaTriX" (54596) ≠ attendu "xatelitte".`
- `whoami` : `Connecté : xatelitte (userId 1214571). Compte attendu : xatelitte.`
  (ou `Compte attendu : (non défini)` si aucun).

## Tests

Le dépôt n'a pas encore de tests. Ajout de `internal/hfr/identity_test.go` avec un serveur HTTP mock
(`httptest`) pour le login et la page profil :

- résolution du userId au login (cookie présent / page parsée / non résolvable → userId vide) ;
- `identityMatches` : match par pseudo (casse mixte), match par userId, non-match ;
- `checkIdentity` : contrainte serveur seule, contrainte appel seule, les deux, aucune (fail-open),
  mismatch sur l'une ou l'autre ;
- garde appelée avant POST : un mismatch n'émet aucune requête de POST.

## Impact / compatibilité

- Rétrocompatible : sans `HFR_EXPECT_LOGIN` ni `--pseudo`, le comportement d'écriture est inchangé,
  enrichi du compte affiché dans le retour.
- Pas de bump de version requis par ce design seul ; à grouper avec le prochain bump (nouveaux outils →
  candidat v1.2.0). Mettre à jour `CHANGELOG.md`, le README (tableaux d'outils + section auth) et l'AGENTS.md.
