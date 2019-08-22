// Copyright (c) of parts are held by the various contributors (see the CLA)
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package opr

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"sort"

	"github.com/pegnet/pegnet/common"
	log "github.com/sirupsen/logrus"
)

const (
	// 0.5%
	GradeBand float64 = 0.005
)

// Avg computes the average answer for the price of each token reported
func Avg(list []*OraclePriceRecord) (avg []float64) {
	avg = make([]float64, len(common.AllAssets))

	// Sum up all the prices
	for _, opr := range list {
		tokens := opr.GetTokens()
		for i, token := range tokens {
			if token.value >= 0 { // Make sure no OPR has negative values for
				avg[i] += token.value // assets.  Simply treat all values as positive.
			} else {
				avg[i] -= token.value
			}
		}
	}
	// Then divide the prices by the number of OraclePriceRecord records.  Two steps is actually faster
	// than doing everything in one loop (one divide for every asset rather than one divide
	// for every asset * number of OraclePriceRecords)  There is also a little bit of a precision advantage
	// with the two loops (fewer divisions usually does help with precision) but that isn't likely to be
	// interesting here.
	numList := float64(len(list))
	for i := range avg {
		avg[i] = avg[i] / numList
	}
	return
}

// CalculateGrade takes the averages and grades the individual OPRs
func CalculateGrade(avg []float64, opr *OraclePriceRecord, band float64) float64 {
	tokens := opr.GetTokens()
	opr.Grade = 0
	for i, v := range tokens {
		if avg[i] > 0 {
			d := (v.value - avg[i]) / avg[i] // compute the difference from the average
			d = ApplyBand(d, GradeBand)      // Apply the band curve
			opr.Grade = opr.Grade + d*d*d*d  // the grade is the sum of the square of the square of the differences
		}
	}
	return opr.Grade
}

// ApplyBand
func ApplyBand(diff float64, band float64) float64 {
	if diff <= band {
		return 0
	}
	return diff - band
}

// GradeMinimum only grades the top 50 honest records. The input must be the records sorted by
// self reported difficulty.
func GradeMinimum(sortedList []*OraclePriceRecord) (graded []*OraclePriceRecord) {
	list := RemoveDuplicateSubmissions(sortedList)
	if len(list) < 10 {
		return nil
	}

	// Find the top 50 with the correct difficulties
	top50 := make([]*OraclePriceRecord, 0)
	for i, opr := range sortedList {
		opr.Difficulty = opr.ComputeDifficulty(opr.Nonce)
		f := binary.BigEndian.Uint64(opr.SelfReportedDifficulty)
		if f != opr.Difficulty {
			log.WithFields(log.Fields{
				"place":     i,
				"entryhash": fmt.Sprintf("%x", opr.EntryHash),
				"id":        opr.FactomDigitalID,
			}).Warnf("Self reported difficulty incorrect Exp %x, found %x", opr.Difficulty, opr.SelfReportedDifficulty)
			continue
		}
		// Honest record
		top50 = append(top50, opr)
		if len(top50) == 50 {
			break // We have enough to grade
		}
	}

	for i := len(top50); i >= 25; i-- {
		avg := Avg(top50[:i])
		// Use the gradeband up to 25 records
		band := GradeBand
		if i < 25 {
			band = 0
		}
		for j := 0; j < i; j++ {
			CalculateGrade(avg, top50[j], band)
		}
		// Because this process can scramble the sorted fields, we have to resort with each pass.
		sort.SliceStable(top50[:i], func(i, j int) bool { return top50[i].Difficulty > top50[j].Difficulty })
		sort.SliceStable(top50[:i], func(i, j int) bool { return top50[i].Grade < top50[j].Grade })
	}

	avg := Avg(top50)
	totald := make(map[string]float64)
	amt := make(map[string]int)
	for i := range avg {
		for _, o := range top50[:] {
			d := (o.GetTokens()[i].value - avg[i]) / avg[i] // compute the difference from the average
			if d < 5 {
				totald[o.GetTokens()[i].code] += d
				amt[o.GetTokens()[i].code] += 1
			}
		}
	}

	avg = Avg(top50[:20])
	oprOut := 0
	for _, o := range top50 {
		for i, token := range o.GetTokens() {
			if math.Abs(token.value-avg[i])/avg[i] > 0.05 {
				if token.code == "JPY" || token.code == "XPT" || token.code == "XPD" || token.code == "INR" || token.code == "CNY" || token.code == "HKD" {
					//continue
				}
				//fmt.Println(o.FactomDigitalID, token.code, token.value, avg[i])
				oprOut++
				break
			}
		}
	}

	//if sortedList[0].Dbht%20 == 0 {
	fmt.Printf("%d -- %d outside band \n", sortedList[0].Dbht, oprOut)
	//for k := range totald {
	//	fmt.Printf("\t%s AvgDiff: %.3f\n", k, totald[k]/float64(amt[k]))
	//}
	//}
	return top50
}

// GradeBlock takes all OPRs in a block, sorts them according to Difficulty, and grades the top 50.
// The top ten graded entries are considered the winners. Returns the top 50 sorted by grade, then the original list
// sorted by difficulty.
func GradeBlock(list []*OraclePriceRecord) (graded []*OraclePriceRecord, sorted []*OraclePriceRecord) {
	list = RemoveDuplicateSubmissions(list)

	if len(list) < 10 {
		return nil, nil
	}

	// Throw away all the entries but the top 50 on pure difficulty alone.
	// Note that we are sorting in descending order.
	sort.SliceStable(list, func(i, j int) bool { return list[i].Difficulty > list[j].Difficulty })

	var topDifficulty []*OraclePriceRecord
	if len(list) > 50 {
		topDifficulty = make([]*OraclePriceRecord, 50)
		copy(topDifficulty[:50], list[:50])
	} else {
		topDifficulty = make([]*OraclePriceRecord, len(list))
		copy(topDifficulty, list)
	}
	for i := len(topDifficulty); i >= 10; i-- {
		avg := Avg(topDifficulty[:i])
		for j := 0; j < i; j++ {
			CalculateGrade(avg, topDifficulty[j], 0)
		}
		// Because this process can scramble the sorted fields, we have to resort with each pass.
		sort.SliceStable(topDifficulty[:i], func(i, j int) bool { return topDifficulty[i].Difficulty > topDifficulty[j].Difficulty })
		sort.SliceStable(topDifficulty[:i], func(i, j int) bool { return topDifficulty[i].Grade < topDifficulty[j].Grade })
	}
	return topDifficulty, list // Return the top50 sorted by grade and then all sorted by difficulty
}

// RemoveDuplicateSubmissions filters out any duplicate OPR (same nonce and OPRHash)
func RemoveDuplicateSubmissions(list []*OraclePriceRecord) []*OraclePriceRecord {
	// nonce+oprhash => exists
	added := make(map[string]bool)
	nlist := make([]*OraclePriceRecord, 0)
	for _, v := range list {
		id := string(append(v.Nonce, v.OPRHash...))
		if !added[id] {
			nlist = append(nlist, v)
			added[id] = true
		}
	}
	return nlist
}

// block data at a specific height
type OprBlock struct {
	OPRs               []*OraclePriceRecord
	GradedOPRs         []*OraclePriceRecord
	Dbht               int64
	TotalNumberRecords int  // We tend to only keep the top 50, it's nice to know how many existed before we cut it
	EmptyOPRBlock      bool // An empty opr block is an eblock that could not totally validate
}

// VerifyWinners takes an opr and compares its list of winners to the winners of previousHeight
func VerifyWinners(opr *OraclePriceRecord, winners []*OraclePriceRecord) bool {
	return true
	for i, w := range opr.WinPreviousOPR {
		if winners == nil && w != "" {
			return false
		}
		if len(winners) > 0 && w != hex.EncodeToString(winners[i].EntryHash[:8]) { // short hash
			return false
		}
	}
	return true
}

func GetRewardFromPlace(place int) int64 {
	if place >= 25 {
		return 0 // There's no participation trophy. Return zero.
	}
	return 200 * 1e8
	switch place {
	case 0:
		return 800 * 1e8 // The Big Winner
	case 1:
		return 600 * 1e8 // Second Place
	default:
		return 450 * 1e8 // Consolation Prize
	}
}
