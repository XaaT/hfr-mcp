# Garde-fou d'identité HFR (issue #32)

**Date** : 2026-06-14
**Issue** : [#32](https://github.com/ForumHFR/hfr-mcp/issues/32) — écritures sous le mauvais compte HFR
**Statut** : design validé (intègre la review Codex gpt-5.5/xhigh du 2026-06-14), prêt pour le plan d'implémentation

## Problème

Les outils d'écriture (`hfr_reply`, `hfr_edit`, `hfr_mp`, `hfr_create_topic`) publient sous le compte HFR
déterminé par la configuration de la session (`HFR_LOGIN` / `hfr.conf`). Ce compte peut dériver
silencieusement d'une session à l'autre, et **rien ne permet de connaître le compte actif avant d'écrire**.
Résultat constaté plusieurs fois : des messages destinés au compte agent sont partis sous un autre compte.

Le forum n'offre aucun outil de suppression : une écriture sous le mauvais compte doit être retirée à la main.
La prévention est donc la seule protection viable.

## Objectifs

1. **Savoir** quel compte est actif avant d'écrire (`hfr_whoami` / `hfr whoami`).
2. **Refuser** d'écrire quand le compte connecté ne correspond pas au compte voulu — déclaré côté serveur
   (`HFR_EXPECT_LOGIN`) et/ou par appel (`--pseudo` / `expect`), et **refuser par défaut** si aucun compte
   attendu n'est déclaré (fail-closed).
3. **Tracer** systématiquement, dans le retour de chaque écriture réussie, le compte effectivement utilisé.

Hors scope : les bugs de l'issue #31 (notif e-mail, cache `hfr_read`), un éventuel `hfr_delete`,
la sélection/bascule de compte au runtime.

## Décisions de design

| Décision | Choix retenu |
|---|---|
| Déclaration du compte attendu | **Défense en profondeur** : barrière serveur `HFR_EXPECT_LOGIN` **et** paramètre par appel (`--pseudo` CLI / `expect` MCP). Les deux contraintes, quand présentes, doivent passer. |
| Critère de comparaison | **Pseudo + userId**, avec **syntaxe typée** : `pseudo:<nom>`, `id:<n>`. Sans préfixe : un attendu purement numérique vise le userId, sinon le pseudo. `"0"` est une contrainte valide (≠ vide). |
| Comportement par défaut (aucun compte attendu) | **Fail-closed** : les écritures sont refusées tant qu'aucun compte attendu n'est déclaré, sauf opt-out explicite `HFR_ALLOW_UNGUARDED_WRITES=1` (qui autorise l'écriture en affichant le compte). |
| Autorité d'identité | Le **cookie `md_user` du jar au moment du POST** (pas une valeur mise en cache au login). |
| Emplacement de la garde | Dans le **`Client` HFR** (`internal/hfr`), via un chemin d'écriture unique. |

## Architecture

La logique vit dans le `Client` HFR, derrière un **chemin d'écriture unique** qui rend atomiques
« résolution de l'identité courante → vérification → POST ». La CLI et le serveur MCP ne font que fournir
les contraintes (`--pseudo` / `expect`, `HFR_EXPECT_LOGIN`) et formater l'`Identity` renvoyée.

### Autorité d'identité (corrige review #1, #2)

Le compte qui signe réellement un POST est porté par le `CookieJar`, pas par un champ mémorisé. La garde lit
donc l'identité **courante** juste avant chaque POST :

- `func (c *Client) currentIdentity() (Identity, error)` : lit le cookie `md_user` du jar (pseudo = autorité),
  associe le `userID` résolu. Erreur si le cookie est absent (session perdue).
- Pour `Edit`, qui exécute un GET (page d'édition) **avant** le POST, la vérification est refaite
  immédiatement avant le POST — pas seulement en tête de fonction.
- Tous les POST d'écriture passent par un helper interne unique (`authenticatedPost`) qui enchaîne
  `currentIdentity` → `checkIdentity` → `doPost` sous le même verrou.

## Composants

### 1. `internal/hfr` — résolution d'identité

- `Client` gagne `userID string`, `expectedLogin string`, `allowUnguarded bool`, `baseURL string`
  (injectable pour les tests, défaut = const actuelle — corrige review #9), et un `sync.Mutex`
  protégeant l'état auth et les écritures (corrige review #8).
- `type Identity struct { Pseudo string; UserID string; Authenticated bool }`.
- **Résolution du `userId`** (corrige review #3) : à l'authentification, parser une **source serveur précise
  sur `/user/editprofil.php`** (déjà chargée par `fetchHashCheck`) — champ/lien exposant l'identifiant ;
  source exacte à figer à l'implémentation. Si le cookie `md_user_id` existe, le valider contre cette source.
  **Pas de userId inventé** : s'il reste irrésolu, `userID` est vide.
- `func (c *Client) Whoami() (Identity, error)` — renvoie l'identité courante (déclenche le login lazy
  côté appelant au préalable).

### 2. `internal/hfr` — garde-fou

```go
// want est une contrainte typée : "pseudo:x", "id:n", ou brut (numérique => userId, sinon pseudo).
func (c *Client) checkIdentity(id Identity, expect string) error {
    constraints := nonEmpty(c.expectedLogin, expect)
    if len(constraints) == 0 {
        if c.allowUnguarded {
            return nil // opt-out explicite
        }
        return ErrNoExpectedAccount // fail-closed
    }
    for _, want := range constraints {
        if !identityMatches(id, want) {
            return ErrIdentityMismatch // message: connecté X (userId N) ≠ attendu want
        }
    }
    return nil
}
```

- `identityMatches` applique la syntaxe typée et **échoue explicitement si l'attendu vise un userId mais que
  `id.UserID` est vide** (corrige review #3, #6).
- Comparaison de pseudo via une fonction `normalize` unique : trim, repli de casse, normalisation Unicode,
  décodage cookie si nécessaire (corrige review #7).
- `""` = pas de contrainte ; `"0"` = contrainte réelle (corrige review #6).

### 3. `internal/hfr` — robustesse du login (corrige review #4)

`Login` ne doit muter `pseudo`/`authed`/`hashCheck`/`userID` **qu'après succès complet** (cookie `md_user`
confirmé **et** `fetchHashCheck` réussi **et** `userID` résolu). En cas d'échec intermédiaire : rollback,
`authed` reste `false`.

### 4. `internal/config` — compte attendu serveur

- Lire `HFR_EXPECT_LOGIN` (env) + `expect_login=` (`hfr.conf`) ; `HFR_ALLOW_UNGUARDED_WRITES` (env).
  L'env prime sur le fichier. Transmis au `Client` à la construction (signature `NewClient` étendue
  ou option fonctionnelle — corrige review #10).

### 5. `internal/mcp` — exposition

- Nouvel outil **`hfr_whoami`** (sans paramètre) : login lazy, renvoie pseudo + userId + compte attendu
  + état du garde (gardé / non gardé).
- Champ optionnel `expect string` ajouté à `ReplyInput`, `EditInput`, `CreateTopicInput`, `MPInput`.
- Le serveur injecte `HFR_EXPECT_LOGIN` / `HFR_ALLOW_UNGUARDED_WRITES` au client au démarrage.
- Les handlers formatent l'`Identity` **retournée par la méthode d'écriture** (pas un `Whoami()` post-hoc —
  corrige review #10).

### 6. `cmd/hfr` — CLI

- Flag global **`--pseudo <login>`** (parsé à côté de `--auth`, aujourd'hui seul géré — corrige review #10) ;
  transmis comme `expect` sur les écritures.
- Nouvelle sous-commande **`hfr whoami`**.
- Lit `HFR_EXPECT_LOGIN` / `HFR_ALLOW_UNGUARDED_WRITES` via la config.

## Signatures (corrige review #10)

```go
func (c *Client) Reply(cat, postId int, content, expect string) (Identity, error)
func (c *Client) Edit(cat, postId, numreponse int, content, expect string) (Identity, error)
func (c *Client) CreateTopic(cat, subcat int, subject, content, expect string) (Identity, error)
func (c *Client) SendMP(dest, subject, content, expect string) (Identity, error)
func (c *Client) Whoami() (Identity, error)
```

## Comportement (flux)

**Écriture** (`reply`/`edit`/`mp`/`new`) :
1. login lazy → résolution de l'identité courante depuis le jar (`currentIdentity`) ;
2. `checkIdentity` contre `HFR_EXPECT_LOGIN` et l'`expect` d'appel, sous verrou ;
   - aucune contrainte et pas d'opt-out → `ErrNoExpectedAccount`, **aucun POST** ;
   - mismatch → `ErrIdentityMismatch`, **aucun POST** ;
3. (Edit) re-vérification immédiatement avant le POST d'édition ;
4. POST, puis retour de l'`Identity` utilisée.

**`whoami`** : login → renvoie l'identité. Lecture seule.

## Formats de message

- Succès : `Message posté sous xatelitte (userId 1214571).`
- Refus mismatch : `écriture refusée : compte connecté "XaTriX" (54596) ≠ attendu "xatelitte".`
- Refus non gardé : `écriture refusée : aucun compte attendu déclaré (pose HFR_EXPECT_LOGIN / --pseudo, ou HFR_ALLOW_UNGUARDED_WRITES=1).`
- `whoami` : `Connecté : xatelitte (userId 1214571). Compte attendu : xatelitte. Garde : active.`

## Erreurs

- `ErrNoExpectedAccount` (code `identity`) — fail-closed, aucun compte attendu.
- `ErrIdentityMismatch` (code `identity`) — compte connecté ≠ attendu, message explicite.

## Tests (corrige review #9)

`internal/hfr` rendu testable via `baseURL`/`http.Client` injectables + serveur `httptest`. Ajout de
`identity_test.go` :

- résolution du userId au login (source présente / cookie incohérent / irrésolvable → userId vide) ;
- `Login` : rollback si `fetchHashCheck` échoue (pas d'état `authed` partiel) ;
- `identityMatches` : `pseudo:` (casse mixte, accents, espaces), `id:`, brut numérique → userId,
  pseudo numérique, attendu userId mais userID vide → échec, `"0"` ≠ `""` ;
- `checkIdentity` : contrainte serveur seule, appel seule, les deux, aucune (fail-closed),
  opt-out `HFR_ALLOW_UNGUARDED_WRITES` ;
- garde avant POST : un mismatch (et le cas non gardé) **n'émet aucune requête de POST** ;
- TOCTOU `Edit` : dérive du cookie `md_user` entre le GET et le POST → refus.

## Impact / compatibilité

- **Changement de comportement** : par défaut, les écritures exigent désormais un compte attendu déclaré
  (fail-closed). Opt-out `HFR_ALLOW_UNGUARDED_WRITES=1` pour le comportement historique.
- Nouveaux outils/flags → candidat **v1.2.0**. Mettre à jour `CHANGELOG.md`, le README (tableaux d'outils
  + section auth + nouvelles variables d'env) et l'`AGENTS.md`.
