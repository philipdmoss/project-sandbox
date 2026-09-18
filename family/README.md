# HHT House of Hanna — Family Tree

Converted from the 91-page HHT 2024 PDF by `parse_hht.py`.

## Files

| File | Description |
|---|---|
| `HHT-House-of-Hanna.ged` | GEDCOM 5.5.1, UTF-8 — the canonical data store |
| `hht-viewer.html` | Self-contained local viewer (open in any browser) |
| `people.json` | Parsed intermediate — one record per person for review/editing |
| `House of Hanna family tree.pdf` | Original source PDF |

## Numbers

- 3,096 INDI records (2,645 tree persons + 451 spouses)
- 869 FAM records (753 with children)
- 11 generation levels; root: John Hanna

## Using the Local Viewer

1. Open `hht-viewer.html` in Chrome/Firefox/Safari — **no server needed**
2. Search for any person by name in the search box
3. Click a node to re-root the chart at that person
4. Switch views: **Descendancy ↓** (children down), **Portrait** (same vertical), **Landscape ←→** (horizontal), **Fan Chart** (ancestor fan)
5. "First Ancestor" button jumps back to John Hanna

> The viewer loads D3.js from CDN (d3js.org) — internet access required on first load.
> After that it works offline if the browser has cached D3.

## Importing the GEDCOM

The `.ged` file is GEDCOM 5.5.1 UTF-8 and imports into:

- **FamilySearch Genealogies** — direct upload at familysearch.org/search/genealogy
- **Ancestry** — My DNA → Manage Test → Upload a GEDCOM
- **MyHeritage** — Add Tree → Import GEDCOM
- **Gramps** (free, local) — File → Import
- **webtrees** — New Tree → Import GEDCOM
- [**topola-viewer**](https://pewu.github.io/topola-viewer) — drag & drop the `.ged` file for polished fan/pedigree/descendancy charts

> FamilySearch's *shared tree* (familysearch.org/tree) does not accept direct GEDCOM import —
> use FamilySearch Genealogies (above) or a certified sync app like RootsMagic.

## Known Decisions (Easy to Correct Per-Person in Any App)

**Sex unset** — Sex is not recorded in the source PDF and was not guessed for ~2,600+ people.
Charts render with neutral styling; set sex in-app for gendered coloring.

**Multi-spouse / unknown child assignment** — When a parent had multiple spouses listed
(e.g. `Edward Nathaniel Hanna` with `Linna Tynes, Jane Cox`), all children are assigned to the
*first* marriage. The source PDF does not indicate which marriage each child belongs to.
Review and correct in your genealogy app.

**Hyphenated surnames** — `Eliza Hanna-Rolle` stores maiden surname `Hanna` as primary SURN
(the blood line identifier) and married surname `Rolle` as `_MARNM` extension tag. Most apps
display this correctly; a few will show only the primary surname.

**Spouse records** — Spouses have names only (no birth/death dates). The source PDF lists
spouse names but no dates for them.
