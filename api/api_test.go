package api_test

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	pb "massnet.org/mass-wallet/api/proto"
)

var (
	body_type = "application/json;charset=utf-8"
)

func BenchmarkAPIServer_TxHistory(b *testing.B) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	for i := 0; i < b.N; i++ {
		res, err := client.Get("https://localhost:9688/v1/transactions/history")
		if err != nil {
			panic(err)
		}
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)

		fmt.Println(string(body))
	}
}

func BenchmarkAPIServer_CreateAddress(b *testing.B) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	tmp := `{"version":0}`
	req := bytes.NewBuffer([]byte(tmp))

	for i := 0; i < b.N; i++ {
		res, err := client.Post("https://localhost:9688/v1/addresses/create", body_type, req)
		if err != nil {
			panic(err)
		}
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		fmt.Println(string(body))
	}
}

func BenchmarkAPIServer_CreateWallet(b *testing.B) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	tmp := `{"passphrase":"123456789"}`
	req := bytes.NewBuffer([]byte(tmp))

	for i := 0; i < b.N; i++ {
		res, err := client.Post("https://localhost:9688/v1/wallets/create", body_type, req)
		if err != nil {
			panic(err)
		}
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		fmt.Println(string(body))
	}
}

// NOTE: set blockchain.FlagNotAddAndRelayTx to true before run benchmark

// goos: darwin
// goarch: amd64
// BenchmarkSendSignedTx                 30          37608562 ns/op         7136948 B/op      54046 allocs/op
// BenchmarkSendSignedTx                 30          38548103 ns/op         7101930 B/op      53788 allocs/op
// BenchmarkSendSignedTx                 50          32267229 ns/op         7109767 B/op      53848 allocs/op
// BenchmarkSendSignedTx                 50          32034102 ns/op         7078380 B/op      53613 allocs/op
// BenchmarkSendSignedTx                 30          35034194 ns/op         7133303 B/op      54019 allocs/op
// BenchmarkSendSignedTx-2              100          19435954 ns/op         7070684 B/op      53552 allocs/op
// BenchmarkSendSignedTx-2              100          19667121 ns/op         7067889 B/op      53536 allocs/op
// BenchmarkSendSignedTx-2              100          19667298 ns/op         7060202 B/op      53481 allocs/op
// BenchmarkSendSignedTx-2              100          19369059 ns/op         7065158 B/op      53515 allocs/op
// BenchmarkSendSignedTx-2              100          19144145 ns/op         7075394 B/op      53591 allocs/op
// BenchmarkSendSignedTx-4              100          12502327 ns/op         7062320 B/op      53499 allocs/op
// BenchmarkSendSignedTx-4              100          12158919 ns/op         7085323 B/op      53667 allocs/op
// BenchmarkSendSignedTx-4              100          12590399 ns/op         7064442 B/op      53507 allocs/op
// BenchmarkSendSignedTx-4              100          12116690 ns/op         7048813 B/op      53394 allocs/op
// BenchmarkSendSignedTx-4              100          12032531 ns/op         7062647 B/op      53498 allocs/op
// 30.833s
func benchmarkSendSignedTx(b *testing.B) {
	file, err := os.Open("./testData/signedTx")
	assert.Nil(b, err)
	defer file.Close()

	ch := make(chan string, 1000)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ch <- scanner.Text()
	}
	// total := len(ch)
	if err := scanner.Err(); err != nil {
		b.Error(err)
	}

	client := getClient()
	format := `{
		"hex":"%s"
	}`

	b.RunParallel(func(bpb *testing.PB) {
		for bpb.Next() {
			tx := <-ch
			defer func() {
				ch <- tx
			}()
			req := bytes.NewBuffer([]byte(fmt.Sprintf(format, tx)))
			res, err := client.Post("https://localhost:9688/v1/transactions/send", body_type, req)
			if err != nil {
				b.Error(err)
			}
			defer res.Body.Close()
			// if res.StatusCode != 200 {
			// 	b.Errorf("response status: %d, %s", res.StatusCode, res.Status)
			// }
		}
	})
}

func testSendSignedTx(t *testing.T) {

	file, err := os.Open("./testData/signedTx")
	assert.Nil(t, err)
	defer file.Close()

	ch := make(chan string, 1000)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ch <- scanner.Text()
	}
	// total := len(ch)
	if err := scanner.Err(); err != nil {
		t.Error(err)
	}

	client := getClient()
	format := `{
		"hex":"%s"
	}`

	i := 0
	for tx := range ch {
		i++
		if i <= 3 {
			continue
		}
		req := bytes.NewBuffer([]byte(fmt.Sprintf(format, tx)))
		res, err := client.Post("https://localhost:9688/v1/transactions/send", body_type, req)
		if err != nil {
			t.Error(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			t.Errorf("response status: %d, %s", res.StatusCode, res.Status)
		}

		resp := &pb.SendRawTransactionResponse{}
		err = unmarshalBody(res.Body, resp)
		assert.Nil(t, err)
		assert.Equal(t, "", resp.TxId)
		break
	}
}

// goos: darwin
// goarch: amd64
// BenchmarkSignTx                2         722555356 ns/op         8173192 B/op      61917 allocs/op
// BenchmarkSignTx-2              3         497923717 ns/op         7839810 B/op      59368 allocs/op
// BenchmarkSignTx-4              5         436932679 ns/op         7548492 B/op      57183 allocs/op
// BenchmarkSignTx-8              5         422998547 ns/op         7559260 B/op      57274 allocs/op
func BenchmarkSignTx(b *testing.B) {
	file, err := os.Open("./testData/rawTx")
	assert.Nil(b, err)
	defer file.Close()

	txs := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txs = append(txs, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		b.Error(err)
	}

	txchan := make(chan string, len(txs))
	for _, rawtx := range txs {
		txchan <- rawtx
	}

	client := getClient()
	format := `{
		"raw_tx":"%s",
		"passphrase": "123456",
		"flags":""
	}`

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rawtx := <-txchan
			defer func() {
				txchan <- rawtx
			}()
			req := bytes.NewBuffer([]byte(fmt.Sprintf(format, rawtx)))
			res, err := client.Post("https://localhost:9688/v1/transactions/sign", body_type, req)
			if err != nil {
				b.Error(err)
			}
			defer res.Body.Close()
			if res.StatusCode != 200 {
				b.Errorf("response status: %d, %s", res.StatusCode, res.Status)
			}
		}
	})
}

func TestPrepareCreateTx(t *testing.T) {
	if testing.Short() {
		t.Skip("")
	}
	client := getClient()

	tmp := `{
		"addresses":["ms1qp3pxf32gjtnv53f4ajclg2ejpu50rpenxuw63u"]
	}`
	req := bytes.NewBuffer([]byte(tmp))
	res, err := client.Post("https://localhost:9688/v1/addresses/utxos", body_type, req)
	if err != nil {
		t.Error(err)
	}
	defer res.Body.Close()
	assert.Nil(t, err)

	resp := &pb.GetUtxoResponse{}
	err = unmarshalBody(res.Body, resp)
	assert.Nil(t, err)

	count := 0
out:
	for _, au := range resp.AddressUtxos {
		fmt.Println(au.Address, len(au.Utxos))
		for _, utxo := range au.Utxos {
			f, err := strconv.ParseFloat(utxo.Amount, 64)
			assert.Nil(t, err)
			if f > 1.0 {
				hex := createRawTx(t, client, utxo)
				assert.NotEmpty(t, hex)

				file, err := os.OpenFile("./testData/rawTx", os.O_APPEND|os.O_WRONLY, 0644)
				assert.Nil(t, err)
				defer file.Close()

				n, err := file.WriteString(hex + "\n")
				assert.True(t, err == nil && n > 0)
				count++
				if count >= 200 {
					break out
				}
			}
		}
	}
}

func createRawTx(t *testing.T, client *http.Client, utxo *pb.UTXO) string {
	format := `{
		"inputs":[
			{
				"tx_id": "%s",
				"vout": %d
			}
		],
		"amounts":{
			"ms1qjjev0kgkkp0vf5yam57x8hw2z9mqlhhvpsv987":"%s"
		},
		"lock_time": 0
	}`

	str := fmt.Sprintf(format, utxo.GetTxId(), utxo.GetVout(), utxo.GetAmount())
	// fmt.Println(str)
	req := bytes.NewBuffer([]byte(str))
	res, err := client.Post("https://localhost:9688/v1/transactions/create", body_type, req)
	if err != nil {
		t.Error(err)
	}
	defer res.Body.Close()
	assert.Nil(t, err)

	resp := &pb.CreateRawTransactionResponse{}
	err = unmarshalBody(res.Body, resp)
	assert.Nil(t, err)

	return resp.GetHex()
}

func TestSignTx(t *testing.T) {
	if testing.Short() {
		t.Skip("")
	}
	file, err := os.Open("./testData/rawTx")
	assert.Nil(t, err)
	defer file.Close()

	fileout, err := os.OpenFile("./testData/signedTx", os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	assert.Nil(t, err)
	defer fileout.Close()

	txs := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txs = append(txs, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Error(err)
	}

	client := getClient()
	format := `{
		"raw_tx":"%s",
		"passphrase": "123456",
		"flags":""
	}`

	count := 0
	for _, rawtx := range txs {
		req := bytes.NewBuffer([]byte(fmt.Sprintf(format, rawtx)))
		res, err := client.Post("https://localhost:9688/v1/transactions/sign", body_type, req)
		if err != nil {
			t.Error(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			t.Errorf("response status: %d, %s", res.StatusCode, res.Status)
		}

		resp := &pb.SignRawTransactionResponse{}
		err = unmarshalBody(res.Body, resp)
		assert.True(t, err == nil && resp.Complete)

		n, err := fileout.WriteString(resp.Hex + "\n")
		assert.True(t, err == nil && n > 0)
		count++
	}
	assert.Equal(t, len(txs), count)
}

func unmarshalBody(body io.ReadCloser, resp interface{}) error {
	u := jsonpb.Unmarshaler{AllowUnknownFields: true}
	return u.Unmarshal(body, resp.(proto.Message))
}

func getClient() *http.Client {
	cert, _ := tls.LoadX509KeyPair("../bin/cert.crt", "../bin/cert.key")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{cert},
		},
	}
	return &http.Client{Transport: tr}
}
