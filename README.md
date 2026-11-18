<p align="center">
<img src="https://user-images.githubusercontent.com/22412567/171680920-6e74b77c-c565-487b-bff1-4f94976ecbe7.png" alt="Artalk" width="100%">
</p>

# Artalk Fork

[![npm version](https://img.shields.io/npm/v/artalk.svg?style=flat-square)](https://www.npmjs.com/package/artalk)
[![npm downloads](https://img.shields.io/npm/dt/artalk.svg?style=flat-square)](https://www.npmjs.com/package/artalk)
[![Docker Pulls](https://img.shields.io/docker/pulls/artalk/artalk-go?style=flat-square)](https://hub.docker.com/r/artalk/artalk-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/artalkjs/artalk/v2.svg)](https://pkg.go.dev/github.com/artalkjs/artalk/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/ArtalkJS/Artalk?style=flat-square)](https://goreportcard.com/report/github.com/ArtalkJS/Artalk)
[![CircleCI](https://img.shields.io/circleci/build/gh/ArtalkJS/Artalk?style=flat-square)](https://circleci.com/gh/ArtalkJS/Artalk/tree/master)
[![Codecov](https://img.shields.io/codecov/c/gh/ArtalkJS/Artalk?style=flat-square)](https://codecov.io/gh/ArtalkJS/Artalk)
[![npm bundle size](https://img.shields.io/bundlephobia/minzip/artalk?style=flat-square)](https://bundlephobia.com/package/artalk)

[Homepage](https://artalk.js.org) • [Documentation](https://artalk.js.org/en/guide/deploy.html) • [Latest Release](https://github.com/ArtalkJS/Artalk/releases) • [Changelog](https://github.com/ArtalkJS/Artalk/blob/master/CHANGELOG.md) • [简体中文](./README.zh.md) • [日本語](./README.ja.md)

Artalk is an intuitive yet feature-rich comment system, ready for immediate deployment into any blog, website, or web application.

- 🍃 Client ~40KB, crafted with pure Vanilla JS, framework-agnostic
- 🍱 Server powered by Golang, offering efficient and lightweight cross-platform performance
- 🐳 One-click deployment via Docker, ensuring ease and speed
- 🌈 Open-source software, self-hosted with privacy as a priority

This repository is a personal-maintained fork of [ArtalkJS/Artalk](https://github.com/ArtalkJS/Artalk). The upstream client API and core comment functionality remain compatible; this fork focuses on backend features, the administration console, and self-hosted deployments.

## Fork and Upstream

- Upstream repository: [ArtalkJS/Artalk](https://github.com/ArtalkJS/Artalk)
- Current repository: [LiuShen-Fork/Artalk](https://github.com/LiuShen-Fork/Artalk)
- Upstream documentation: [artalk.js.org](https://artalk.js.org)
- `master` is based on `upstream/master`; fork-specific commits are kept here and synchronized manually.
- The fork does not publish a separate npm client package. The frontend client remains based on the upstream package, while the server and administration console are maintained here.

### Current state of this fork

- Base: `upstream/master` (2026-11-14, `d9b62cba`).
- History: the fork's customizations are split by feature across a handful of commits on top of upstream — server capabilities, administration frontend, documentation and CI — so a later rebase resolves conflicts in a handful of files in one pass.
- Size: roughly 330 files changed, +23.3k / −9.3k. The server accounts for ~125 files (~+12.4k) and `ui/artalk-sidebar` for ~122 files, while the visitor-facing `ui/artalk` client changes only 18 files by roughly 110 lines — the comment widget itself stays close to upstream.
- Administration console: `ui/artalk-sidebar` is a React 19 console built on HeroUI v3, Tailwind CSS v4, TanStack Router and TanStack Query. It talks to the same `/api/v2` endpoints as before, runs at 430px inside the widget's iframe, keeps every locale of the widget client and adds no hard-coded deployment details. See [Administration console](#administration-console).
- Accounts: upstream already provides password and social login. What this fork adds is what happens after login — self-service password changes, binding and unbinding third-party identities, and a security page — plus login and sign-up integrated into the sidebar so ordinary users can manage their own profile without the admin console.
- Compatibility: the public API and the widget's events and options match upstream; every addition is opt-in and off by default.

### Hard fork and data compatibility

- Starting with **v2.10.1** this repository is a hard fork: upstream changes are still merged by hand, but the fork’s schema and configuration keep moving ahead of upstream.
- **Upgrading from upstream to this fork is additive.** Drop in a binary built from this repository and start it with your existing `artalk.yml`: on start it creates the tables and columns the fork added (`moderation_logs`, `security_events`, `ip_bans` and `users.is_ai_assistant`), and de-duplicates the auth-identity table before adding its unique indexes. Nothing existing is rewritten or dropped. The new configuration sections (`security`, `console`, `http.trusted_proxies`, …) are optional — left empty they keep the old behaviour, and with `security.enabled: false` the upstream limiter still runs. The administration console is replaced wholesale and holds no data.
- **Data compatibility is not guaranteed.** The fork adds its own tables and columns (`moderation_logs`, `security_events`, `ip_bans`, `users.is_ai_assistant`) and migrates them automatically on start. A database written by this fork is not guaranteed to be readable by an upstream release, and the reverse only holds for the Artalk tables the two share. Export a backup before upgrading across the fork point.
- **Export always works, in the upstream format.** The console’s *Transfer* page produces `.artrans` archives with the untouched upstream export handler, so the file format is identical to upstream Artalk. Import accepts backups from upstream and from this fork, so data can move in either direction.

## Differences from Upstream

### AI comment moderation

The fork adds optional asynchronous AI moderation to the existing moderation pipeline. It supports OpenAI Responses, OpenAI Chat Completions, and DeepSeek JSON Output, with built-in JSON Schema output, editable prompts, configurable thinking mode, and configurable behavior when the provider request fails.

### AI comment assistant

If page text cannot be fetched, the assistant posts a fixed network-check message; if the AI provider request fails, it posts a fixed backend-interface message. The underlying error remains available in the AI assistant log.

The fork provides an optional assistant that answers comments when mentioned with its configured name or when a user directly replies to an assistant comment. It uses up to 20 earlier messages from the same thread, page content, and the current comment as context, preserving chronological order so provider prefix caching can reuse the stable prompt prefix. It avoids duplicate replies in nested threads and supports OpenAI Responses, OpenAI Chat Completions, DeepSeek JSON Output, and Claude Messages. `content_selector` and `exclude_selectors` limit page extraction without failing when a selector is absent. `daily_limit` (default 40) and `user_hourly_limit` (default 5) prevent abuse; over-limit requests save a fixed local reply without calling the provider. `reply_to_pending` controls whether comments awaiting moderation can trigger a reply, and the prompt is editable as a multiline field in the administration console.

The assistant is stored as a dedicated user identity, separate from its configurable email address. This keeps avatar customization available while preventing reply notifications from being sent to the assistant itself. Replies generated by the assistant still send normal notifications to the original commenter. Assistant comments are also shown in a separate `AI Assistant` section above the regular comment list for troubleshooting, and the administration comments page provides an `AI Assistant` filter while keeping them in `All`. Existing databases are upgraded automatically; existing users and comments remain unchanged.

### Comment interception

Synchronous local keyword interception is available under `moderator.intercept`. Enter comma-separated keywords to match against both the nickname and comment content. A match returns HTTP `400`, prevents persistence and notifications, and records the rejected content in the moderation log. The feature is disabled by default and uses a small local list to avoid external dictionary and performance overhead.

### Security hardening

An opt-in `security` block (off by default) adds rate limiting, banning and attack records on top of the existing captcha action counter:

- Requests are counted per IP and globally over minute, hour and day windows; `0` means unlimited.
- A trip can require the captcha (reusing the existing `need_captcha` response, so the official widget needs no change), reject with `429`, or ban the IP with `403`.
- IPs that keep tripping inside a configured window are auto-banned; manual and CIDR bans live in `ip_bans`, config-file bans are read-only in the console.
- Blocked requests are stored in `security_events` (IP region, path, UA, site, account, counts), pruned after a retention period, and aggregated into a report by kind, day, top IPs and top paths.
- Admins and whitelisted IPs are not counted; only write methods are; the legacy `LimiterGuard` yields when security is enabled so nothing is counted or challenged twice.
- The console's 安全防护 page owns both the `security` and `captcha` blocks — limits, exempt paths, whitelist and blacklist, the captcha type and every provider's keys, bans, report and records — and edits them in place (a YAML patch that preserves comments, applied without a restart). 系统设置 no longer lists those two blocks.

### Administration console

`ui/artalk-sidebar` is the page the widget opens in an iframe. This fork rebuilds it from scratch on a current stack instead of the previous Vue components:

- React 19, HeroUI v3 (`@heroui/react` + `@heroui/styles`), Tailwind CSS v4 with an oklch token theme, TanStack Router (hash history, so it keeps working inside the iframe) and TanStack Query for server state.
- Pages keep their previous behavior — dashboard, comments, moderation, pages, users, sites, transfer, settings, security and account — and share one shell: a left sidebar on desktop that narrows to an icon rail on medium screens and becomes a bottom tab bar on phones, so the same build works inside the 430px iframe and in a full browser window.
- The console has two faces. An admin switches between the management pages and "my pages" (their own comments, their own account) at the top of the sidebar; everybody else only ever sees the second one, and the menu, the routes and the data scope all follow that choice.
- The login screen is part of the console: password sign-in, sign-up with an email verification code, password reset, and third-party providers when the backend enables them. Provider buttons are disabled with an explanation when the console's own origin is not in the trusted domain list, since the callback would be rejected.
- The comment list is not reimplemented: the page mounts the widget's own list element, so rendering, replying, moderation and AI-assistant behavior stay exactly as the public client does them.
- Every request still goes through the widget's API client (`ctx.getApi()`), so tokens, guest credentials and error messages behave as before; the backend was not changed for this rewrite.
- One component per job: every list pages through the same pager (page numbers with gaps, first/last jumps, a live total and a page-size picker that remembers the choice), and every comment card takes its look from one stylesheet (`src/styles/comments.css`) instead of the widget defaults.
- The console is brandable from its own **Console** page (name, icon, description, custom CSS/JS), stored in the `console` section of the config file and served by `GET/PUT /api/v2/sidebar/console`; the name and description also appear on the sign-in screen.
- **My account** is the single page for the profile form, the credential change, linked identities and sign-out; the avatar in the corner opens a small menu that jumps to that page, switches between the management and visitor views, and signs out. Inside the widget iframe the host page's close button can cover the avatar, but every one of those actions is also reachable from the navigation and from that page.
- All nine locales of the console are kept in sync (`src/i18n-en.ts` defines the message schema), and it contains no deployment-specific values.

### Other fork-specific changes

- Moderation and AI assistant logs, review actions, dashboard pages, and a Chinese-default administration console.
- Generic OAuth 2.0 login and OIDC SSO token exchange.
- Direct Lsky Pro image hosting uploads, while Upgit remains available.
- Separate IPv6 ip2region database configuration.
- Fork-specific CI/CD and Docker publishing workflows, with releases triggered manually.

The upstream comment API and basic capabilities remain available, including Markdown, nested comments, multi-site management, notifications, CAPTCHA, keyword moderation, database options, Docker deployment, localization, dark mode, lazy loading, lightbox support, and IP region display.

## Syncing Upstream

~~~bash
git remote add upstream https://github.com/ArtalkJS/Artalk.git
git fetch upstream
git rebase upstream/master
~~~

After rebasing, review conflicts in the moderation pipeline, configuration metadata, administration console, and release workflows. Run the backend tests and administration console build before publishing.

## Features

<!-- prettier-ignore-start -->

<!-- features -->
* [Sidebar](https://artalk.js.org/guide/frontend/sidebar.html): Quick management, intuitive browsing
* [Social Login](https://artalk.js.org/guide/frontend/auth.html): Fast login via social accounts
* [Email Notification](https://artalk.js.org/guide/backend/email.html): Various sending methods, email templates
* [Diverse Push](https://artalk.js.org/guide/backend/admin_notify.html): Multiple push methods, notification templates
* [Site Notification](https://artalk.js.org/guide/frontend/sidebar.html): Red dot marks, mention list
* [Captcha](https://artalk.js.org/guide/backend/captcha.html): Various verification types, frequency limits
* [Comment Moderation](https://artalk.js.org/guide/backend/moderator.html): Content detection, spam interception
* [Image Upload](https://artalk.js.org/guide/backend/img-upload.html): Custom upload, supports image hosting
* [Markdown](https://artalk.js.org/guide/intro.html): Supports Markdown syntax
* [Emoji Pack](https://artalk.js.org/guide/frontend/emoticons.html): Compatible with OwO, quick integration
* [Multi-Site](https://artalk.js.org/guide/backend/multi-site.html): Site isolation, centralized management
* [Admin](https://artalk.js.org/guide/backend/multi-site.html): Password verification, badge identification
* [Page Management](https://artalk.js.org/guide/frontend/sidebar.html): Quick view, one-click title navigation
* [Page View Statistics](https://artalk.js.org/guide/frontend/pv.html): Easily track page views
* [Hierarchical Structure](https://artalk.js.org/guide/frontend/config.html#nestmax): Nested paginated list, infinite scroll
* [Comment Voting](https://artalk.js.org/guide/frontend/config.html#vote): Upvote or downvote comments
* [Comment Sorting](https://artalk.js.org/guide/frontend/config.html#listsort): Various sorting options, freely selectable
* [Comment Search](https://artalk.js.org/guide/frontend/sidebar.html): Quick comment content search
* [Comment Pinning](https://artalk.js.org/guide/frontend/sidebar.html): Pin important messages
* [View Author Only](https://artalk.js.org/guide/frontend/config.html): Show only the author's comments
* [Comment Jump](https://artalk.js.org/guide/intro.html): Quickly jump to quoted comment
* [Auto Save](https://artalk.js.org/guide/frontend/config.html): Content loss prevention
* [IP Region](https://artalk.js.org/guide/frontend/ip-region.html): Display user's IP region
* [Data Migration](https://artalk.js.org/guide/transfer.html): Free migration, quick backup
* [Image Lightbox](https://artalk.js.org/guide/frontend/lightbox.html): Quick integration of image lightbox
* [Image Lazy Load](https://artalk.js.org/guide/frontend/img-lazy-load.html): Lazy load images, optimize experience
* [Latex](https://artalk.js.org/guide/frontend/latex.html): Integrate Latex formula parsing
* [Night Mode](https://artalk.js.org/guide/frontend/config.html#darkmode): Switch to night mode
* [Extension Plugin](https://artalk.js.org/develop/plugin.html): Create more possibilities
* [Multi-Language](https://artalk.js.org/guide/frontend/i18n.html): Switch between multiple languages
* [Command Line](https://artalk.js.org/guide/backend/config.html): Command line operation management
* [API Documentation](https://artalk.js.org/http-api.html): Provides OpenAPI format documentation
* [Program Upgrade](https://artalk.js.org/guide/backend/update.html): Version check, one-click upgrade
<!-- /features -->

<!-- prettier-ignore-end -->

## Installation

Deploy Artalk Server with Docker in one step:

```bash
docker run -d \
    --name artalk \
    -p 8080:23366 \
    -v $(pwd)/data:/data \
    -e "TZ=America/New_York" \
    -e "ATK_LOCALE=en" \
    -e "ATK_SITE_DEFAULT=Artalk Blog" \
    -e "ATK_SITE_URL=https://example.com" \
    artalk/artalk-go
```

Integrate Artalk Client into your webpage:

<!-- prettier-ignore-start -->

```ts
Artalk.init({
  el:      '#Comments',
  site:    'Artalk Blog',
  server:  'https://artalk.example.com',
  pageKey: '/2018/10/02/hello-world.html'
})
```

<!-- prettier-ignore-end -->

We offer various installation methods, including binary files, go install, and package managers for Linux distributions.

[**Learn More →**](https://artalk.js.org/en/guide/deploy.html)

## For Developers

Pull requests are welcome!

See [Development](https://artalk.js.org/en/develop/) and [Contributing](./CONTRIBUTING.md) for information on working with the codebase, getting a local development setup, and contributing changes.

## Contributors

Your contributions enrich the open-source community, fostering learning, inspiration, and innovation. We deeply value your involvement. Thank you for being a vital part of our community! 🥰

[![](https://contrib.rocks/image?repo=ArtalkJS/Artalk)](https://github.com/ArtalkJS/Artalk/graphs/contributors)

## Supporters

[![Stargazers repo roster for @ArtalkJS/Artalk](https://reporoster.com/stars/ArtalkJS/Artalk)](https://github.com/ArtalkJS/Artalk/stargazers)

## Repobeats Analytics

![Alt](https://repobeats.axiom.co/api/embed/a9fc9191ac561bc5a8ee2cddc81e635ecaebafb6.svg 'Repobeats analytics image')

## Stargazers over time

<a href="https://trendshift.io/repositories/6290" target="_blank"><img src="https://trendshift.io/api/badge/repositories/6290" alt="ArtalkJS%2FArtalk | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>

[![Stargazers over time](https://starchart.cc/ArtalkJS/Artalk.svg)](https://starchart.cc/ArtalkJS/Artalk)

## License

[MIT](./LICENSE)

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FArtalkJS%2FArtalk.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2FArtalkJS%2FArtalk?ref=badge_shield)

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FArtalkJS%2FArtalk.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2FArtalkJS%2FArtalk?ref=badge_large)
