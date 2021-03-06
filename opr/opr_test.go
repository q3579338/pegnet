// Copyright (c) of parts are held by the various contributors (see the CLA)
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package opr_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"github.com/FactomProject/btcutil/base58"
	"github.com/pegnet/pegnet/common"
	. "github.com/pegnet/pegnet/opr"
)

func TestOPR_JSON_Marshal(t *testing.T) {
	LX.Init(0x123412341234, 25, 256, 5)
	opr := NewOraclePriceRecord()

	opr.Difficulty = 1
	opr.Grade = 1
	//opr.Nonce = base58.Encode(LX.Hash([]byte("a Nonce")))
	//opr.ChainID = base58.Encode(LX.Hash([]byte("a chainID")))
	opr.Dbht = 1901232
	opr.WinPreviousOPR = [10]string{
		base58.Encode(LX.Hash([]byte("winner number 1"))),
		base58.Encode(LX.Hash([]byte("winner number 2"))),
		base58.Encode(LX.Hash([]byte("winner number 3"))),
		base58.Encode(LX.Hash([]byte("winner number 4"))),
		base58.Encode(LX.Hash([]byte("winner number 5"))),
		base58.Encode(LX.Hash([]byte("winner number 6"))),
		base58.Encode(LX.Hash([]byte("winner number 7"))),
		base58.Encode(LX.Hash([]byte("winner number 8"))),
		base58.Encode(LX.Hash([]byte("winner number 9"))),
		base58.Encode(LX.Hash([]byte("winner number 10"))),
	}
	opr.CoinbasePNTAddress = "PNT4wBqpZM9xaShSYTABzAf1i1eSHVbbNk2xd1x6AkfZiy366c620f"
	opr.FactomDigitalID = "minerone"
	opr.Assets["PNT"] = 2
	opr.Assets["USD"] = 20
	opr.Assets["EUR"] = 200
	opr.Assets["JPY"] = 11
	opr.Assets["GBP"] = 12
	opr.Assets["CAD"] = 13
	opr.Assets["CHF"] = 14
	opr.Assets["INR"] = 15
	opr.Assets["SGD"] = 16
	opr.Assets["CNY"] = 17
	opr.Assets["HKD"] = 18
	opr.Assets["XAU"] = 19
	opr.Assets["XAG"] = 101
	opr.Assets["XPD"] = 1012
	opr.Assets["XPT"] = 10123
	opr.Assets["XBT"] = 10124
	opr.Assets["ETH"] = 10125
	opr.Assets["LTC"] = 10126
	opr.Assets["XBC"] = 10127
	opr.Assets["FCT"] = 10128

	v, _ := json.Marshal(opr)
	fmt.Println("len of entry", len(string(v)), "\n\n", string(v))
	opr2 := NewOraclePriceRecord()
	err := json.Unmarshal(v, &opr2)
	if err != nil {
		t.Fail()
	}
	v2, _ := json.Marshal(opr2)
	fmt.Println("\n\n", string(v2))
	if string(v2) != string(v) {
		t.Error("JSON is different")
	}
}

func rstring(len int) string {
	r := make([]byte, len)
	rand.Read(r)
	return string(r)
}

func genJSONOPR() *OraclePriceRecord {
	opr := new(OraclePriceRecord)
	opr.CoinbaseAddress = rstring(56)
	opr.Dbht = rand.Int31()
	for i := range opr.WinPreviousOPR {
		opr.WinPreviousOPR[i] = rstring(8)
	}
	opr.FactomDigitalID = rstring(25)
	opr.Assets = make(OraclePriceRecordAssetList)
	for _, a := range common.AllAssets {
		opr.Assets[a] = rand.Float64() * 5000
	}
	return opr
}

func BenchmarkJSONMarshal(b *testing.B) {
	b.StopTimer()
	InitLX()
	data := make([]*OraclePriceRecord, 0)
	for i := 0; i < b.N; i++ {
		data = append(data, genJSONOPR())
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(data[i])
	}
}

func BenchmarkSingleOPRHash(b *testing.B) {
	b.StopTimer()
	InitLX()
	data := make([][]byte, 0)
	for i := 0; i < b.N; i++ {
		json, _ := json.Marshal(genJSONOPR())
		data = append(data, json)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		LX.Hash(data[i])
	}
}

func BenchmarkSingleSha256(b *testing.B) {
	b.StopTimer()
	InitLX()
	data := make([][]byte, 0)
	for i := 0; i < b.N; i++ {
		json, _ := json.Marshal(genJSONOPR())
		data = append(data, json)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		sha256.Sum256(data[i])
	}
}
func BenchmarkComputeDifficulty(b *testing.B) {
	b.StopTimer()
	data := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		data[i] = make([]byte, 37) // 32 oprhash + "average" 5 byte nonce
		rand.Read(data[i])
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ComputeDifficulty(data[i][0:32], data[i][32:])
	}
}
