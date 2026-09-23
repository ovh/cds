# Specification: Theme

## Overview

The CDS UI ships two skins, **light** and **night**. The theme in force drives the body class, the
ng-zorro dark stylesheet, the Monaco editor theme, and the asset folder icons are fetched from — so
every component that cares only ever needs to know which of the two is currently applied.

What the user picks, however, is not one of those two skins but a **mode**, and one of the three
modes defers the decision to the operating system.

---

## 1 — Modes

| Mode | Applied theme |
|---|---|
| `auto` (default) | whatever the browser reports as the system preference |
| `light` | light, whatever the system says |
| `night` | night, whatever the system says |

The mode is a user preference: it is stored with the other preferences and survives reloads. The
applied theme is derived from it and never stored as a choice of its own.

`auto` is the default, so a user who never opens the theme selector gets the skin their desktop
already asked for. Picking `light` or `night` forces that skin until the user goes back to `auto` —
a forced theme is a decision, and the system preference does not override it.

---

## 2 — Following the System

The system preference is read from the browser's `prefers-color-scheme` media query at startup and
watched for changes afterwards. A user switching their desktop to dark at sunset sees the UI
follow immediately, without a reload, as long as the mode is `auto`.

The observed system preference is tracked as part of the preferences state rather than read on
demand, so the theme stays a plain derived value that any component can select synchronously.

---

## 3 — Selecting a Mode

The navbar carries a dedicated theme button, next to the help one. Its icon shows the **applied**
theme — a sun for light, a moon for night — so in `auto` it follows the system along with the rest
of the UI.

Clicking it opens a small popup holding the whole selector: a three-way choice between **Auto**,
**Light** and **Dark**, which closes as soon as a mode is picked. Unlike the button, the selector
reflects the stored mode: a user on `auto` with a dark desktop sees a moon on the button and
*Auto* selected inside the popup.
