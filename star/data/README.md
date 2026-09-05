# Embedded star catalog

`stars.csv` is a curated subset of the **HYG database** (Hipparcos-Yale-Gliese),
version 4.4, compiled by David Nash / Astronomy Nexus and distributed at
<https://codeberg.org/astronexus/hyg>.

## License

The HYG database is licensed under the
[Creative Commons Attribution-ShareAlike 4.0 International License][cc]
(CC BY-SA 4.0). This subset remains under that license; see the repository-root
`NOTICE` file for the full attribution.

[cc]: https://creativecommons.org/licenses/by-sa/4.0/

## Subset and columns

To keep the compiled binary small, only a subset of the catalog is embedded:
every star with apparent magnitude **≤ 6.5** (the naked-eye sky) plus every star
carrying a **proper name**, regardless of magnitude. The Sun (HYG id 0) is
excluded. Rows are sorted by ascending magnitude (brightest first).

Only the columns the library exposes are kept. Right ascension is stored in
**degrees** (the HYG source stores hours); positions are J2000.

| Column   | Meaning                                             |
| -------- | --------------------------------------------------- |
| `proper` | Proper / IAU name (may be empty)                    |
| `bf`     | Bayer / Flamsteed designation (may be empty)        |
| `con`    | IAU constellation abbreviation                      |
| `ra_deg` | Right ascension, degrees, J2000                     |
| `dec`    | Declination, degrees, J2000                         |
| `dist`   | Distance, parsecs (0 when unknown)                  |
| `mag`    | Apparent visual magnitude                           |
| `hip`    | Hipparcos catalog number (may be empty)             |
| `hd`     | Henry Draper catalog number (may be empty)          |
| `hr`     | Harvard Revised / Bright Star number (may be empty) |
| `gl`     | Gliese-Jahreiss identifier (may be empty)           |
