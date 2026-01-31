# Sporeprint vs. fpcalc

## Compatibility

Sporeprint has been tested against a mixture of 12,145 files.

In 4 occasions, sporeprint disagreed with fpcalc (stdin mode), but agreed with fpcalc (direct).

In 20 occasions, sporeprint disagreed with fpcalc (direct), but agreed with fpcalc (stdin).

In 10 cases, sporeprint disagreed with both.

For the "direct" divergences, this suggests that fpcalc may still be doing something on the files when
converting them itself that we do not know about and do not reproduce correctly
with our ffmpeg stance above.

The stdin cases are more head-scratching: is fpcalc still doing some transformation,
even on prepared streams?

In any case, the discrepancies in the result are minor and localized,
and the fingerprints should still resolve to the same acoustid.

That is about 0.28% of cases on our sample collection then.

So, conservatively, you may expect less than 1% divergence with fpcalc, using
the provided ffmpeg transformation, of which most should resolve to the same acoustid.

## Performance

Both tools perform about the same, which is unsurprising given Chromaprint FFT bears the grunt
of the cost.

You are looking at 40 minutes for these 12+k files.
