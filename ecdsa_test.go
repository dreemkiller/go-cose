package cose

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"reflect"
	"testing"
)

func TestI2OSP(t *testing.T) {
	tests := []struct {
		name    string
		x       *big.Int
		buf     []byte
		want    []byte
		wantErr bool
	}{
		{
			name:    "negative int",
			x:       big.NewInt(-1),
			buf:     make([]byte, 2),
			wantErr: true,
		},
		{
			name:    "integer too large #1",
			x:       big.NewInt(1),
			buf:     make([]byte, 0),
			wantErr: true,
		},
		{
			name:    "integer too large #2",
			x:       big.NewInt(256),
			buf:     make([]byte, 0),
			wantErr: true,
		},
		{
			name:    "integer too large #3",
			x:       big.NewInt(1 << 24),
			buf:     make([]byte, 3),
			wantErr: true,
		},
		{
			name: "zero length string",
			x:    big.NewInt(0),
			buf:  make([]byte, 0),
			want: []byte{},
		},
		{
			name: "zero length string with nil buffer",
			x:    big.NewInt(0),
			buf:  nil,
			want: nil,
		},
		{
			name: "I2OSP(0, 2)",
			x:    big.NewInt(0),
			buf:  make([]byte, 2),
			want: []byte{0x00, 0x00},
		},
		{
			name: "I2OSP(1, 2)",
			x:    big.NewInt(1),
			buf:  make([]byte, 2),
			want: []byte{0x00, 0x01},
		},
		{
			name: "I2OSP(255, 2)",
			x:    big.NewInt(255),
			buf:  make([]byte, 2),
			want: []byte{0x00, 0xff},
		},
		{
			name: "I2OSP(256, 2)",
			x:    big.NewInt(256),
			buf:  make([]byte, 2),
			want: []byte{0x01, 0x00},
		},
		{
			name: "I2OSP(65535, 2)",
			x:    big.NewInt(65535),
			buf:  make([]byte, 2),
			want: []byte{0xff, 0xff},
		},
		{
			name: "I2OSP(1234, 5)",
			x:    big.NewInt(1234),
			buf:  make([]byte, 5),
			want: []byte{0x00, 0x00, 0x00, 0x04, 0xd2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := I2OSP(tt.x, tt.buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("I2OSP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got := tt.buf; !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("I2OSP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOS2IP(t *testing.T) {
	tests := []struct {
		name string
		x    []byte
		want *big.Int
	}{
		{
			name: "zero length string",
			x:    []byte{},
			want: big.NewInt(0),
		},
		{
			name: "OS2IP(I2OSP(0, 2))",
			x:    []byte{0x00, 0x00},
			want: big.NewInt(0),
		},
		{
			name: "OS2IP(I2OSP(1, 2))",
			x:    []byte{0x00, 0x01},
			want: big.NewInt(1),
		},
		{
			name: "OS2IP(I2OSP(255, 2))",
			x:    []byte{0x00, 0xff},
			want: big.NewInt(255),
		},
		{
			name: "OS2IP(I2OSP(256, 2))",
			x:    []byte{0x01, 0x00},
			want: big.NewInt(256),
		},
		{
			name: "OS2IP(I2OSP(65535, 2))",
			x:    []byte{0xff, 0xff},
			want: big.NewInt(65535),
		},
		{
			name: "OS2IP(I2OSP(1234, 5))",
			x:    []byte{0x00, 0x00, 0x00, 0x04, 0xd2},
			want: big.NewInt(1234),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OS2IP(tt.x); tt.want.Cmp(got) != 0 {
				t.Errorf("OS2IP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func generateTestECDSAKey(t *testing.T) *ecdsa.PrivateKey {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey() error = %v", err)
	}
	return key
}

func Test_customCurveKeySigner(t *testing.T) {
	// https://github.com/veraison/go-cose/issues/59
	pCustom := *elliptic.P256().Params()
	pCustom.Name = "P-custom"
	pCustom.BitSize /= 2
	key, err := ecdsa.GenerateKey(&pCustom, rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey() error = %v", err)
	}
	testSignVerify(t, AlgorithmES256, key, false)
}

func Test_ecdsaKeySigner(t *testing.T) {
	key := generateTestECDSAKey(t)
	testSignVerify(t, AlgorithmES256, key, false)
}

func Test_ecdsaCryptoSigner(t *testing.T) {
	wrappedKey := struct {
		crypto.Signer
	}{
		Signer: generateTestECDSAKey(t),
	}
	testSignVerify(t, AlgorithmES256, wrappedKey, true)
}

func testSignVerify(t *testing.T, alg Algorithm, key crypto.Signer, isCryptoSigner bool) {
	// set up signer
	signer, err := NewSigner(alg, key)
	if err != nil {
		t.Fatalf("NewSigner() error = %v", err)
	}
	if isCryptoSigner {
		if _, ok := signer.(*ecdsaCryptoSigner); !ok {
			t.Fatalf("NewSigner() type = %v, want *ecdsaCryptoSigner", reflect.TypeOf(signer))
		}
	} else {
		if _, ok := signer.(*ecdsaKeySigner); !ok {
			t.Fatalf("NewSigner() type = %v, want *ecdsaKeySigner", reflect.TypeOf(signer))
		}
	}
	if got := signer.Algorithm(); got != alg {
		t.Fatalf("Algorithm() = %v, want %v", got, alg)
	}

	// sign / verify round trip
	// see also conformance_test.go for strict tests.
	digest, err := alg.computeHash([]byte("hello world"))
	if err != nil {
		t.Fatalf("Algorithm.computeHash() error = %v", err)
	}
	sig, err := signer.Sign(rand.Reader, digest)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	verifier, err := NewVerifier(alg, key.Public())
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}
	if err := verifier.Verify(digest, sig); err != nil {
		t.Fatalf("Verifier.Verify() error = %v", err)
	}
}

func Test_ecdsaVerifier_Verify_Success(t *testing.T) {
	// generate key
	alg := AlgorithmES256
	key := generateTestECDSAKey(t)

	// generate a valid signature
	digest, sig := signTestData(t, alg, key)

	// set up verifier
	verifier, err := NewVerifier(alg, key.Public())
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}
	if _, ok := verifier.(*ecdsaVerifier); !ok {
		t.Fatalf("NewVerifier() type = %v, want *ecdsaVerifier", reflect.TypeOf(verifier))
	}
	if got := verifier.Algorithm(); got != alg {
		t.Fatalf("Algorithm() = %v, want %v", got, alg)
	}

	// verify round trip
	if err := verifier.Verify(digest, sig); err != nil {
		t.Fatalf("ecdsaVerifier.Verify() error = %v", err)
	}
}

func Test_ecdsaVerifier_Verify_AlgorithmMismatch(t *testing.T) {
	// generate key
	alg := AlgorithmES256
	key := generateTestECDSAKey(t)

	// generate a valid signature
	digest, sig := signTestData(t, alg, key)

	// set up verifier with a different algorithm
	verifier := &ecdsaVerifier{
		alg: AlgorithmES512,
		key: &key.PublicKey,
	}

	// verification should fail on algorithm mismatch
	if err := verifier.Verify(digest, sig); err != ErrVerification {
		t.Fatalf("ecdsaVerifier.Verify() error = %v, wantErr %v", err, ErrVerification)
	}
}

func Test_ecdsaVerifier_Verify_KeyMismatch(t *testing.T) {
	// generate key
	alg := AlgorithmES256
	key := generateTestECDSAKey(t)

	// generate a valid signature
	digest, sig := signTestData(t, alg, key)

	// set up verifier with a different key / new key
	key = generateTestECDSAKey(t)
	verifier := &ecdsaVerifier{
		alg: alg,
		key: &key.PublicKey,
	}

	// verification should fail on key mismatch
	if err := verifier.Verify(digest, sig); err != ErrVerification {
		t.Fatalf("ecdsaVerifier.Verify() error = %v, wantErr %v", err, ErrVerification)
	}
}

func Test_ecdsaVerifier_Verify_InvalidSignature(t *testing.T) {
	// generate key
	alg := AlgorithmES256
	key := generateTestECDSAKey(t)

	// generate a valid signature with a tampered one
	digest, sig := signTestData(t, alg, key)
	tamperedSig := make([]byte, len(sig))
	copy(tamperedSig, sig)
	tamperedSig[0]++

	// set up verifier with a different algorithm
	verifier := &ecdsaVerifier{
		alg: alg,
		key: &key.PublicKey,
	}

	// verification should fail on invalid signature
	tests := []struct {
		name      string
		signature []byte
	}{
		{
			name:      "nil signature",
			signature: nil,
		},
		{
			name:      "empty signature",
			signature: []byte{},
		},
		{
			name:      "incomplete signature",
			signature: sig[:len(sig)-2],
		},
		{
			name:      "tampered signature",
			signature: tamperedSig,
		},
		{
			name:      "too many signature bytes",
			signature: append(sig, 0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := verifier.Verify(digest, tt.signature); err != ErrVerification {
				t.Errorf("ecdsaVerifier.Verify() error = %v, wantErr %v", err, ErrVerification)
			}
		})
	}
}
