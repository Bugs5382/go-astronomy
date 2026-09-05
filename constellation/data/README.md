# Embedded constellation data

## `boundaries.csv`

The IAU constellation boundary table established by Eugène Delporte (1930) and
rearranged for position lookup by Nancy G. Roman (1987), distributed as VizieR
catalog [VI/42](https://vizier.cds.unistra.fr/viz-bin/VizieR?-source=VI/42). Each
row is one southern boundary arc at the **B1875.0** equinox:

| Column       | Meaning                                                 |
| ------------ | ------------------------------------------------------- |
| `ra_low_h`   | Lower right-ascension bound of the arc, hours (B1875.0) |
| `ra_up_h`    | Upper right-ascension bound of the arc, hours (B1875.0) |
| `de_low_deg` | Lower (southern) declination of the arc, degrees        |
| `con`        | Three-letter IAU constellation abbreviation             |

Rows are in the source order (descending declination), which the Roman lookup
algorithm relies on.

## `constellations.csv`

The 88 IAU constellations with their Latin name and genitive form.

| Column     | Meaning                                       |
| ---------- | --------------------------------------------- |
| `abbrev`   | Three-letter IAU abbreviation (e.g. `Ori`)    |
| `name`     | Latin nominative name (e.g. `Orion`)          |
| `genitive` | Latin genitive name (e.g. `Orionis`)          |

See the repository-root `NOTICE` for full attribution.
