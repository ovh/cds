package sssa

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strings"
)

var prime *big.Int

/**
 * Returns a random number from the range (0, prime-1) inclusive
**/
func random() *big.Int {
	result := big.NewInt(0).Set(prime)
	result = result.Sub(result, big.NewInt(1))
	result, _ = rand.Int(rand.Reader, result)
	return result
}

/**
 * Converts a byte array into an a 256-bit big.Int, arraied based upon size of
 * the input byte; all values are right-padded to length 256, even if the most
 * significant bit is zero.
**/
func splitByteToInt(secret []byte) []*big.Int {
	hex_data := hex.EncodeToString(secret)
	count := int(math.Ceil(float64(len(hex_data)) / 64.0))

	result := make([]*big.Int, count)

	for i := 0; i < count; i++ {
		if (i+1)*64 < len(hex_data) {
			result[i], _ = big.NewInt(0).SetString(hex_data[i*64:(i+1)*64], 16)
		} else {
			data := strings.Join([]string{hex_data[i*64:], strings.Repeat("0", 64-(len(hex_data)-i*64))}, "")
			result[i], _ = big.NewInt(0).SetString(data, 16)
		}
	}

	return result
}

/**
 * Converts an array of big.Ints to the original byte array, removing any
 * least significant nulls
**/
func mergeIntToByte(secret []*big.Int) []byte {
	var hex_data = ""
	for i := range secret {
		tmp := fmt.Sprintf("%x", secret[i])
		hex_data += strings.Join([]string{strings.Repeat("0", (64 - len(tmp))), tmp}, "")
	}

	result, _ := hex.DecodeString(hex_data)
	result = bytes.TrimRight(result, "\x00")

	return result
}

/**
 * Evauluates a polynomial with coefficients specified in reverse order:
 * evaluatePolynomial([a, b, c, d], x):
 * 		returns a + bx + cx^2 + dx^3
**/
func evaluatePolynomial(polynomial []*big.Int, value *big.Int) *big.Int {
	last := len(polynomial) - 1
	var result *big.Int = big.NewInt(0).Set(polynomial[last])

	for s := last - 1; s >= 0; s-- {
		result = result.Mul(result, value)
		result = result.Add(result, polynomial[s])
		result = result.Mod(result, prime)
	}

	return result
}

/**
 * inNumbers(array, value) returns boolean whether or not value is in array
**/
func inNumbers(numbers []*big.Int, value *big.Int) bool {
	for n := range numbers {
		if numbers[n].Cmp(value) == 0 {
			return true
		}
	}

	return false
}

/**
 * Returns the big.Int number base10 in base64 representation; note: this is
 * not a string representation; the base64 output is exactly 256 bits long
**/
func toBase64(number *big.Int) string {
	hexdata := fmt.Sprintf("%x", number)
	for i := 0; len(hexdata) < 64; i++ {
		hexdata = "0" + hexdata
	}
	bytedata, success := hex.DecodeString(hexdata)
	if success != nil {
		fmt.Println("Error!")
		fmt.Println("hexdata: ", hexdata)
		fmt.Println("bytedata: ", bytedata)
		fmt.Println(success)
	}
	return base64.URLEncoding.EncodeToString(bytedata)
}

/**
 * Returns the number base64 in base 10 big.Int representation; note: this is
 * not coming from a string representation; the base64 input is exactly 256
 * bits long, and the output is an arbitrary size base 10 integer.
 *
 * Returns -1 on failure
**/
func fromBase64(number string) *big.Int {
	bytedata, err := base64.URLEncoding.DecodeString(number)
	if err != nil {
		return big.NewInt(-1)
	}

	hexdata := hex.EncodeToString(bytedata)
	result, ok := big.NewInt(0).SetString(hexdata, 16)
	if ok == false {
		return big.NewInt(-1)
	}

	return result
}

/**
 * Computes the multiplicative inverse of the number on the field prime; more
 * specifically, number * inverse == 1; Note: number should never be zero
**/
func modInverse(number *big.Int) *big.Int {
	copy := big.NewInt(0).Set(number)
	copy = copy.Mod(copy, prime)
	pcopy := big.NewInt(0).Set(prime)
	x := big.NewInt(0)
	y := big.NewInt(0)

	copy.GCD(x, y, pcopy, copy)

	result := big.NewInt(0).Set(prime)

	result = result.Add(result, y)
	result = result.Mod(result, prime)
	return result
}
