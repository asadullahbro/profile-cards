package main

import "math"

// rank grades a profile the way github-readme-stats does, so the grade on the
// stats card matches the old one: each measure is compared with a median, the
// weighted result is a percentile, and the percentile picks the letter.
func rank(commits, prs, issues, reviews, stars, followers int) (level string, percentile float64) {
	exp := func(x float64) float64 { return 1 - math.Pow(2, -x) }
	logn := func(x float64) float64 { return x / (1 + x) }
	const total = 2 + 3 + 1 + 1 + 4 + 1
	r := 1 - (2*exp(float64(commits)/250)+
		3*exp(float64(prs)/50)+
		1*exp(float64(issues)/25)+
		1*exp(float64(reviews)/2)+
		4*logn(float64(stars)/50)+
		1*logn(float64(followers)/10))/total
	percentile = r * 100
	levels := []string{"S", "A+", "A", "A-", "B+", "B", "B-", "C+", "C"}
	thresholds := []float64{1, 12.5, 25, 37.5, 50, 62.5, 75, 87.5, 100}
	level = levels[len(levels)-1]
	for i, t := range thresholds {
		if percentile <= t {
			level = levels[i]
			break
		}
	}
	return level, percentile
}
