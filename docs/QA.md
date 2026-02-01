# Sporeprint vs. fpcalc

## Compatibility

Mass fingerprint comparison across 10,972 audio files using three methods:
- **FD**: fpcalc direct (built-in decoder)
- **FS**: fpcalc stdin (ffmpeg pipe, s16le 11025Hz mono)
- **SP**: sporeprint (ffmpeg pipe, s16le 11025Hz mono)

### Results

| #  | File                                                                                                                                                                                     | FD   | FS  | SP  | Agreement           | Determinism |
|----|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------|-----|-----|---------------------|----|
| 1  | `/Volumes/Anisotope/gill/2000-08-07 - Supersmell/[2000-08-07-CD-FIX_LABEL-159 014-2-601215901429]/01-15 Psychoflute.mp3`                                                                 | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 2  | `/Volumes/Anisotope/gill/2000-08-07 - Supersmell/[2000-08-07-CD-FIX_LABEL-159 014-2-601215901429]/02-15 Tijuana 65.mp3`                                                                  | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 3  | `/Volumes/Anisotope/gill/2000-08-07 - Supersmell/[2000-08-07-CD-FIX_LABEL-159 014-2-601215901429]/03-15 Dreamtime in Memphis.mp3`                                                        | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 4  | `/Volumes/Anisotope/gill/2000-08-07 - Supersmell/[2000-08-07-CD-FIX_LABEL-159 014-2-601215901429]/04-15 Erotic City.mp3`                                                                 | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 5  | `/Volumes/Anisotope/gill/2000-08-07 - Supersmell/[2000-08-07-CD-FIX_LABEL-159 014-2-601215901429]/05-15 Baby Elephant Walk.mp3`                                                          | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 6  | `/Volumes/Anisotope/gill/2000-08-07 - Supersmell/[2000-08-07-CD-FIX_LABEL-159 014-2-601215901429]/06-15 The Cape and the Crown (Sha La La).mp3`                                          | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 7  | `/Volumes/Anisotope/gill/2000-08-07 - Supersmell/[2000-08-07-CD-FIX_LABEL-159 014-2-601215901429]/07-15 One More Try.mp3`                                                                | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 8  | `/Volumes/Anisotope/gill/2000-08-07 - Supersmell/[2000-08-07-CD-FIX_LABEL-159 014-2-601215901429]/08-15 Under Control.mp3`                                                               | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 9  | `/Volumes/Anisotope/gill/2000-08-07 - Supersmell/[2000-08-07-CD-FIX_LABEL-159 014-2-601215901429]/09-15 No. 9.mp3`                                                                       | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 10 | `/Volumes/Anisotope/gill/2000-08-07 - Supersmell/[2000-08-07-CD-FIX_LABEL-159 014-2-601215901429]/10-15 Meeting Isaac (For Your Love).mp3`                                               | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 11 | `/Volumes/Anisotope/gill/2000-08-07 - Supersmell/[2000-08-07-CD-FIX_LABEL-159 014-2-601215901429]/11-15 Glass Tiger.mp3`                                                                 | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 12 | `/Volumes/Anisotope/gill/2000-08-07 - Supersmell/[2000-08-07-CD-FIX_LABEL-159 014-2-601215901429]/12-15 Riders on the Storm.mp3`                                                         | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 13 | `/Volumes/Anisotope/gill/2000-08-07 - Supersmell/[2000-08-07-CD-FIX_LABEL-159 014-2-601215901429]/13-15 Draw Me Something, Mr. Jack Brewer.mp3`                                          | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 14 | `/Volumes/Anisotope/gill/2000-08-07 - Supersmell/[2000-08-07-CD-FIX_LABEL-159 014-2-601215901429]/14-15 Sphincterludi.mp3`                                                               | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 15 | `/Volumes/Anisotope/gill/Jazz shitty compilations/Mingus, Charles/2010 - Eight Classic Albums/[2010-CD-Real Gone Jazz-RGJCD221-5036408118226]/disc 2-4/06-10 East Coasting (take 4).m4a` | OK   | OK  | OK  | FS~SP, FD~FS, FD=SP | yes |
| 16 | `/Volumes/Anisotope/gill/jazz.lossless/Russell, George/- _Ezz-thetic_/[-FIX_MEDIA-FIX_LABEL-]/05-12 Night Sound.m4a`                                                                     | OK   | OK  | OK  | FS~SP, FD=FS, FD~SP | yes |
| 17 | `/Volumes/Anisotope/gill/old.lossless/Bailey, Derek & Ruins/1995-07-18 - Saisoro/[1995-07-18-CD-Tzadik-TZ 7205]/01-07 Yaginbo.m4a`                                                       | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 18 | `/Volumes/Anisotope/gill/old.lossless/Murray, Keith/1999-01-12 - It's a Beautiful Thing/[1999-01-12-CD-Jive-7243 8 47164 2 4-724384716424]/16-19 High as Hell.m4a`                       | OK   | OK  | OK  | FS~SP, FD~FS, FD=SP | FD:BROKEN, FS:BROKEN |
| 19 | `/Volumes/Anisotope/gill/sweep.zorn/Ambitious Lovers/1988-09-21 - Greed/[-FIX_MEDIA-FIX_LABEL-]/10-13 Para Não Contrariar Você.mp3`                                                      | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 20 | `/Volumes/Anisotope/gill/sweep.zorn/Brody, Paul/2007-05-22 - For the Moment/[2007-05-22-CD-Tzadik-TZ 8118-702397811824]/01-10 Warsaw.mp3`                                                | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 21 | `/Volumes/Anisotope/gill/sweep.zorn/Brody, Paul/2007-05-22 - For the Moment/[2007-05-22-CD-Tzadik-TZ 8118-702397811824]/02-10 Too Low.mp3`                                               | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 22 | `/Volumes/Anisotope/gill/sweep.zorn/Brody, Paul/2007-05-22 - For the Moment/[2007-05-22-CD-Tzadik-TZ 8118-702397811824]/03-10 Bartoki.mp3`                                               | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 23 | `/Volumes/Anisotope/gill/sweep.zorn/Brody, Paul/2007-05-22 - For the Moment/[2007-05-22-CD-Tzadik-TZ 8118-702397811824]/04-10 Serendipity.mp3`                                           | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 24 | `/Volumes/Anisotope/gill/sweep.zorn/Brody, Paul/2007-05-22 - For the Moment/[2007-05-22-CD-Tzadik-TZ 8118-702397811824]/05-10 Sit Down.mp3`                                              | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 25 | `/Volumes/Anisotope/gill/sweep.zorn/Brody, Paul/2007-05-22 - For the Moment/[2007-05-22-CD-Tzadik-TZ 8118-702397811824]/06-10 Good-Bye for Jetzt.mp3`                                    | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 26 | `/Volumes/Anisotope/gill/sweep.zorn/Brody, Paul/2007-05-22 - For the Moment/[2007-05-22-CD-Tzadik-TZ 8118-702397811824]/07-10 Dukovinia.mp3`                                             | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 27 | `/Volumes/Anisotope/gill/sweep.zorn/Brody, Paul/2007-05-22 - For the Moment/[2007-05-22-CD-Tzadik-TZ 8118-702397811824]/08-10 For the Moment.mp3`                                        | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 28 | `/Volumes/Anisotope/gill/sweep.zorn/Brody, Paul/2007-05-22 - For the Moment/[2007-05-22-CD-Tzadik-TZ 8118-702397811824]/10-10 Guitar.mp3`                                                | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 29 | `/Volumes/Anisotope/gill/sweep.zorn/Brötzmann Clarinet Project/1987 - Berlin Djungle/[2004-CD-Atavistic-UMS_ALP246CD]/01-02 What a Day, Part 1.mp3`                                      | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 30 | `/Volumes/Anisotope/gill/sweep.zorn/Company 91/1991 - Volume 2/[1991-CD-Incus Records-CD17]/06-08 YR_AB_VM.mp3`                                                                          | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 31 | `/Volumes/Anisotope/gill/sweep.zorn/Cracow Klezmer Band, The/2006-05-23 - Balan_ Book of Angels, Volume 5/[2006-05-23-CD-Tzadik-TZ 7358-702397735823]/01-08 Zuriel.mp3`                  | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 32 | `/Volumes/Anisotope/gill/sweep.zorn/Hanrahan, Kip/1986 - Desire Develops An Edge/[1986-CD-american clavé-]/06-17 (Don't Complicate) The Life (La Vie).mp3`                               | OK   | OK  | OK  | FD=FS=SP            | FD:BROKEN |
| 33 | `/Volumes/Anisotope/gill/sweep.zorn/Hemophiliac/2002-06 - Hemophiliac/[2002-06-CD-Tzadik-TZ 0001]/disc 1-2/01-08 Skin Eruptions.mp3`                                                     | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 34 | `/Volumes/Anisotope/gill/sweep.zorn/Hemophiliac/2002-06 - Hemophiliac/[2002-06-CD-Tzadik-TZ 0001]/disc 2-2/01-10 Gotu Kola.mp3`                                                          | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 35 | `/Volumes/Anisotope/gill/sweep.zorn/Mystic Fugu Orchestra/1995-07-18 - Zohar/[1995-07-18-CD-Tzadik-TZ 7106]/06-08 Goniff Dance.mp3`                                                      | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |
| 36 | `/Volumes/Anisotope/gill/sweep.zorn/Patton, John/1993 - Blue Planet Man/[1993-CD-Bellaphon-KICJ 168]/02-08 Funky Mama.mp3`                                                               | FAIL | OK  | OK  | FD:FAIL, FS=SP      | yes |

### Legend

- **FD** = fpcalc direct (built-in decoder)
- **FS** = fpcalc stdin (ffmpeg decodes, pipes raw PCM to fpcalc)
- **SP** = sporeprint (ffmpeg decodes, pipes raw PCM to sporeprint)
- `=` identical fingerprint strings
- `~` different raw strings, but `sporeprint compare` confirms match
- `!=` compare rejects (no match)
- **Determinism**: each tool was run 3 times on each file. `yes` = same result every run. `BROKEN` = fpcalc produces different fingerprints across runs on the same input.

### Totals

- Files tested: 10,972
- Incidents: 36 / 10,972 (0.33%)
- fpcalc direct decoder crashes: 32
- Fingerprint variance (all pairwise compare-match): 4
- True sporeprint mismatches: 0
- sporeprint determinism failures: 0

### fpcalc broken determinism

fpcalc produces different fingerprints across identical runs on 2 files. A correct fingerprinting algorithm given the same input must produce the same output. This is broken code in fpcalc's built-in decoder.

| File | Broken mode |
|------|-------------|
| `/Volumes/Anisotope/gill/old.lossless/Murray, Keith/1999-01-12 - It's a Beautiful Thing/[1999-01-12-CD-Jive-7243 8 47164 2 4-724384716424]/16-19 High as Hell.m4a` | FD and FS both produce different fingerprints across runs |
| `/Volumes/Anisotope/gill/sweep.zorn/Hanrahan, Kip/1986 - Desire Develops An Edge/[1986-CD-american clavé-]/06-17 (Don't Complicate) The Life (La Vie).mp3` | FD produces different fingerprints across runs (FS is stable) |

sporeprint produced identical output on all 3 runs for every file tested.

## Performance

Both tools perform about the same, which is unsurprising given Chromaprint FFT bears the grunt
of the cost.

You are looking at 40 minutes for these 12+k files.
