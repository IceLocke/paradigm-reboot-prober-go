#!/usr/bin/env python3
"""Summarize `fitting evaluate` JSON; votes never enter the production fitter."""
import argparse
import json
import math
import random
import statistics as st


def mean(values):
    return st.mean(values) if values else None


def correlation(x, y):
    if len(x) < 2:
        return None
    mx, my = st.mean(x), st.mean(y)
    denom = math.sqrt(sum((v-mx)**2 for v in x) * sum((v-my)**2 for v in y))
    return sum((a-mx)*(b-my) for a, b in zip(x, y))/denom if denom else None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("evaluation")
    args = parser.parse_args()
    with open(args.evaluation) as f:
        data = json.load(f)
    charts = {str(c["id"]): c for c in data["charts"]}
    votes = {str(v["chart"]): v for v in data["votes"]}
    results = data["results"]
    fits = {name: {c: r["FittingLevel"] for c, r in rows.items() if r["FittingLevel"] is not None}
            for name, rows in results.items()}
    common = sorted(set(votes).intersection(*(set(f) for f in fits.values())), key=int)
    # Split by song rather than difficulty, so two charts of one song stay together.
    train = [c for c in common if charts[c]["song_id"] % 5 != 0]
    heldout = [c for c in common if charts[c]["song_id"] % 5 == 0]

    def metrics(fit, ids):
        errors = [abs(fit[c]-votes[c]["level"]) for c in ids]
        pred = [fit[c]-charts[c]["level"] for c in ids]
        actual = [votes[c]["level"]-charts[c]["level"] for c in ids]
        directional = [(p, a) for p, a in zip(pred, actual) if abs(a) >= 0.03]
        return dict(n=len(ids), mae=mean(errors), rmse=math.sqrt(mean([e*e for e in errors])) if errors else None,
                    vote_weighted_mae=sum(e*votes[c]["count"] for e,c in zip(errors,ids))/sum(votes[c]["count"] for c in ids) if ids else None,
                    residual_correlation=correlation(pred,actual),
                    direction_n=len(directional), direction_accuracy=mean([p*a > 0 for p,a in directional]))

    report = dict(matched_votes=len(votes), common_votes=len(common), train_n=len(train), heldout_n=len(heldout))
    x = [fits["calibrated_unscaled_b20"][c]-charts[c]["level"] for c in train]
    y = [votes[c]["level"]-charts[c]["level"] for c in train]
    report["training_least_squares_gain"] = sum(a*b for a,b in zip(x,y))/sum(a*a for a in x) if any(x) else None
    report["models"] = {}
    fits["official"] = {c: row["level"] for c,row in charts.items()}
    for name, fit in fits.items():
        dev = [value-charts[c]["level"] for c,value in fit.items()]
        report["models"][name] = dict(published=len(fit), mean_official_deviation=mean(dev),
            positive=sum(v>1e-9 for v in dev), negative=sum(v < -1e-9 for v in dev),
            common=metrics(fit,common), train=metrics(fit,train), heldout=metrics(fit,heldout),
            bands={str(b):dict(n=len(ds), bias=mean(ds)) for b in range(1,19)
                   if (ds := [v-charts[c]["level"] for c,v in fit.items() if int(charts[c]["level"])==b])})
    # Paired song bootstrap: uncertainty for the small retrospective holdout.
    rng = random.Random(20260906)
    songs = sorted({charts[c]["song_id"] for c in heldout})
    improvements = []
    for _ in range(2000 if songs else 0):
        sample = [c for song in rng.choices(songs,k=len(songs)) for c in heldout if charts[c]["song_id"]==song]
        improvements.append(mean([abs(charts[c]["level"]-votes[c]["level"])-abs(fits["calibrated_b20"][c]-votes[c]["level"]) for c in sample]))
    report["holdout_mae_improvement_over_official_bootstrap_95pct"] = [sorted(improvements)[i] for i in (50,1949)] if improvements else None
    report["examples"] = [dict(title=charts[c]["title"],official=charts[c]["level"],vote=votes[c]["level"],
                              old=fits["legacy_b20"][c],new=fits["calibrated_b20"][c])
                          for c in sorted(common,key=lambda c:abs(votes[c]["level"]-charts[c]["level"]),reverse=True)[:12]]
    print(json.dumps(report,ensure_ascii=False,indent=2))


if __name__ == "__main__":
    main()
