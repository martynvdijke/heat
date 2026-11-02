# Google Play submission

This is the operator and developer reference for the Play Console declarations.
The app is `xyz.vandijke.heat`, a companion for the board game *Heat: Pedal to
the Metal*. It talks to a self-hosted HEAT server over HTTPS.

Update this file whenever a permission, SDK, endpoint or server data field
changes. The Data safety form is a legal declaration, not a formality.

## Store listing facts

| Field | Value |
| --- | --- |
| App name | HEAT — Pedal to the Metal companion |
| Default language | English |
| App or game | App |
| Free or paid | Free |
| Ads | No |
| In-app purchases | No |
| Privacy policy URL | `https://heat.vandijke.xyz/privacy` |
| Account deletion URL | `https://heat.vandijke.xyz/account/delete` |
| In-app deletion path | Settings → Delete racer identity / Delete admin account |
| Target audience | 16+; not designed for children |
| Content rating | Expected Everyone / PEGI 3 (no violence, sex, profanity, drugs, gambling, or scary content) |
| Target API level | 36 (Android 16) via the Flutter SDK; release manifest keeps `usesCleartextTraffic="false"` |

Both legal URLs must load **without signing in** and must explain what happens
to data, not merely exist. `/account/delete` does both. The app also links to all
three pages from Settings.

## App access for reviewers

The app has no built-in demo server. All functionality is served by the instance
at `PUBLIC_BASE_URL`. There are two roles:

- **Spectator / standings** works without a login.
- **Racer identity** needs a racer login (email link or player token).
- **Race director** needs an admin username and password.

Recommended App access declaration: *"All functionality is available with the
provided test credentials."* Supply the following through Play Console → App
access, never in this repository:

1. A test admin account (`username` / `password`) on the review instance.
2. A test racer record with a long-lived player token, so the reviewer can use
   the **Player Token** login mode without access to an email inbox.
   Issue one with `POST /api/player/login` for the test racer and paste the
   returned token into App access.
3. If an email-link test is preferred, set the reviewer's address on a racer and
   request a link from `/me.html`; links expire after 15 minutes, so this is a
   less reliable review path. Do not rely on it as the only test route.

If the instance is invite-only, mark the restriction and still provide the test
admin credentials. Reviewers reject declarations that leave no way to reach the
account features.

## Data safety

**Does the app collect or share required user data types?** Yes.

The app itself has no analytics, crash-reporting or advertising SDK. The
following is collected by the app and its self-hosted backend. "Shared" follows
Google's definition: transferred to a third party for their own use. Service
providers acting on the operator's instructions (such as an SMTP relay) are not
counted as sharing, but are listed in the privacy notice.

| Data type | Collected | Shared | Purpose | Optional | Ephemeral |
| --- | --- | --- | --- | --- | --- |
| Personal info → Name | Yes — racer display name | No | Account management, app functionality, public leaderboard | No (a racer profile is created by the operator) | No |
| Personal info → Email address | Yes — racer sign-in email; admin recovery email | No | Account management, authentication | Yes for admin recovery; required for racer email login | No |
| Personal info → User IDs | Yes — racer id, admin account id | No | Account management, app functionality | No | No |
| Personal info → Other info | Yes — Telegram chat id, username and first name, only when a racer links Telegram via the bot | No | Account management, notifications | Yes | No |
| Photos and videos → Photos | Yes — optional racer profile picture | No | App functionality (public profile) | Yes | No |
| App activity → Other user-generated content | Yes — car name/colour, team, profile fields | No | App functionality, public leaderboard | No | No |
| App activity → Other actions | Yes — race results, round scores, upgrades, achievements, statistics | No | App functionality, public results | No | No |
| Device or other IDs | Yes — random player-session token and device name; session IP address in server logs | No | Authentication, security, app functionality | Player session is created on player login | No |
| Location | No | — | — | — | — |
| Financial info | No | — | — | — | — |
| Health and fitness | No | — | — | — | — |
| Messages | No (the Android app has no messaging feature) | — | — | — | — |
| Contacts | No | — | — | — | — |
| Calendar | No | — | — | — | — |
| Web browsing | No | — | — | — | — |
| In-app search history | No | — | — | — | — |
| Installed apps | No | — | — | — | — |
| Audio / files / docs | No | — | — | — | — |

Notes that matter during review:

- **Racer names and results are public.** Leaderboards, public race state and
  the public API expose them. That is why App activity and Name are "collected"
  and visible, even though there is no social feed.
- **Profile pictures are optional.** Administrators create racer records; the
  mobile app does not upload photos.
- **Telegram is not used by the Android app.** It is a separate bot integration
  on the server. Disclose it because the backend stores the link, and answer the
  "Other info" row accordingly.
- **No advertising, no sale of data, no profiling with legal effect.** ELO and
  standings are descriptive statistics.
- **Encryption in transit.** Release builds reject cleartext HTTP; the service
  is served over TLS. Local development can enable cleartext only in the debug
  manifest.
- **Backups.** `android:allowBackup="false"`, `fullBackupContent="false"` and
  `res/xml/data_extraction_rules.xml` keep the session cookie, player token and
  server URL out of cloud backup and device transfer.

## Data deletion

Play requires three things, all of which are implemented:

1. **A public URL** that explains deletion: `https://heat.vandijke.xyz/account/delete`.
2. **An in-app route**: Settings → Delete racer identity (or Delete admin
   account). The user types `DELETE`; the app calls `DELETE /api/me` or
   `DELETE /api/account`, then clears local credentials.
3. **A fallback** when the user cannot sign in: the contact address configured
   in `LEGAL_CONTACT_EMAIL`, described on the same page.

What deletion covers:

- Racer identity: racer row, emails, Telegram link and prefs, player sessions,
  upgrades, legend abilities, achievements, stats, driver-share links, gameplay
  records. Historical `race_results` and `round_snapshot_scores` are detached and
  renamed to "Deleted racer".
- Admin account: admin row, password-reset tokens, every admin web session.
  Race data is untouched.
- Backups and logs are retained for their configured periods; the public page
  says so.

The last remaining admin cannot be deleted. The API returns HTTP 409 and the
public page explains why.

## Content rating questionnaire

Answer the IARC questionnaire truthfully for the current build:

| Question | Answer |
| --- | --- |
| Violence, blood, fear | None |
| Sexuality, nudity | None |
| Language / profanity | None |
| Controlled substances | None |
| Gambling, simulated gambling | None |
| User-generated content shared with others | Yes — display names, optional profile pictures and race results are visible to other users |
| Users can interact online | Yes — shared leaderboards and live race state; no chat or messaging |
| Shares user location | No |
| Allows purchases | No |
| Unrestricted web access | No — the app opens the privacy, terms and deletion pages in the system browser |

Expected outcome: Everyone / PEGI 3. Re-check after adding any chat, photo
upload or external browser feature.

## Target audience and children

- Target age group: 16+.
- Not designed for children; the privacy notice states that accounts are not
  intended for under-16s.
- No ads, no ad personalisation, no Families programme enrolment.
- The Play Console "Target audience" answer must not include under-13 age
  bands.

## Permissions and manifest

The release manifest requests only `android.permission.INTERNET`. Verify before
each upload:

- `android:usesCleartextTraffic="false"` in `src/main/AndroidManifest.xml`;
  only `src/debug/AndroidManifest.xml` overrides it to `true`.
- `android:allowBackup="false"`, `android:fullBackupContent="false"` and
  `android:dataExtractionRules="@xml/data_extraction_rules"`.
- No new dangerous permissions (camera, location, contacts, storage) without a
  matching in-app gesture and an update to the Data safety form.
- `targetSdk` resolves to API 36 through the Flutter SDK. Check the merged
  manifest of the release AAB if Flutter is upgraded.

## Release checklist

1. Set `LEGAL_CONTROLLER_NAME`, `LEGAL_CONTACT_EMAIL`, `LEGAL_CONTACT_ADDRESS`
   and `LEGAL_HOSTING_REGION` on the production server; restart it.
2. Open `/privacy`, `/terms` and `/account/delete` in a signed-out browser and
   confirm the contact details and processor list match reality.
3. Confirm `/account/delete` is reachable at the exact URL entered in Play
   Console and returns HTTP 200 without a session.
4. Build the release AAB and inspect the merged manifest for cleartext and
   backup settings.
5. Re-run `go test ./...` and `flutter test` in `mobile/`.
6. Update the Data safety table above if any endpoint or integration changed.
7. Upload, complete the IARC questionnaire, and provide App access credentials
   through Play Console only.
