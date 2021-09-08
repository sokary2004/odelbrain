package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"hash"
	"io/ioutil"
	"log"
	"math/big"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gosuri/uilive"
	"github.com/haltingstate/secp256k1-go"
	"github.com/jbenet/go-base58"

	//boom "github.com/tylertreat/BoomFilters"
	boom "github.com/bits-and-blooms/bloom/v3"
	"golang.org/x/crypto/ripemd160"

	_ "github.com/mattn/go-sqlite3"
)

var addressList, wordList []string

type Wallet struct {
	base58BitcoinAddress string
	RIPEM160             string
	privateKey           string
	passphrase           string
}

var botStartElapsed time.Time

//var start time.Time
var writer *uilive.Writer
var total uint64 = 0
var totalFound uint64 = 0

//var sqliteDatabase *sql.DB
var walletList_opt *string = flag.String("wallet", "wallets.txt", "wallets.txt")
var walletInsert_opt *bool = flag.Bool("walletinsert", false, "true")
var phraseCount_opt *int = flag.Int("phrasecount", 12, "12")
var input_opt *string = flag.String("input", "phrases.txt", "phrases.txt")
var output_opt *string = flag.String("output", "bingo.txt", "bingo.txt")
var thread_opt *int = flag.Int("thread", 8, "1")

var sbf *boom.BloomFilter

func main() {
	flag.Parse()
	fmt.Println("Wallet:", *walletList_opt)
	fmt.Println("Wallet Insert:", *walletInsert_opt)
	fmt.Println("Phrase Count:", *phraseCount_opt)
	fmt.Println("Input:", *input_opt)
	fmt.Println("Output:", *output_opt)
	fmt.Println("Thread:", *thread_opt)
	wordListRead, _ := ioutil.ReadFile(*input_opt)
	bytesRead, _ := ioutil.ReadFile(*walletList_opt)
	file_content := string(bytesRead)
	addressList = strings.Split(strings.Replace(file_content, "\r\n", "\n", -1), "\n")
	file_content = string(wordListRead)
	wordList = strings.Split(strings.Replace(file_content, "\r\n", "\n", -1), "\n")
	//fmt.Println(wordList)
	//fmt.Println(addressList)
	addressCount := len(addressList)
	fmt.Println("Total Wallet:", uint(addressCount))
	sbf = boom.NewWithEstimates(uint(addressCount), 0.01) //0.00000000000000000001

	for _, address := range addressList {
		sbf.Add([]byte(address))
	}
	fmt.Println("Wallets Loaded") //Loaded

	// if sbf.Test([]byte(AddressToRIPEM160(`15Jp2rPA5zbEZV4rwCSLufreJWyJUfyA58`))) {
	// 	fmt.Println("contains b")
	// }

	// if *walletInsert_opt {
	// 	createDB()
	// }
	// sqliteDatabase, _ = sql.Open("sqlite3", "database.db") // Open the created SQLite File
	// defer sqliteDatabase.Close()                           // Defer Closing the database
	// if *walletInsert_opt {
	// 	InsertWalletToDB()
	// }
	go Counter() //Stat
	// var wg sync.WaitGroup
	// for i := 1; i <= *thread_opt; i++ {
	// 	wg.Add(1)
	// 	go Brute(i, &wg)
	// }
	// wg.Wait()

	var wg sync.WaitGroup

	/*
	 * Tell the 'wg' WaitGroup how many threads/goroutines
	 *  that are about to run concurrently.
	 */
	wg.Add(*thread_opt)

	fmt.Println("Threads Starting")
	for i := 0; i < *thread_opt; i++ {

		/*
		 * Spawn a thread for each iteration in the loop.
		 * Pass 'i' into goroutine's function
		 * in order to make sure each goroutine use a different value for 'i'
		 */
		go func(i int) {
			// At the end of the goroutine, tell the WaitGroup that another thread has completed
			defer wg.Done()
			Brute(i, &wg)
			fmt.Printf("i: %v\n", i)
		}(i)
	}
	wg.Wait()
	fmt.Println("Threads Started")

	//Close Function
	sig := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sig
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()
	fmt.Println("awaiting signal")
	<-done
	fmt.Println("exiting")
	//Close Function
}

func init() {
	botStartElapsed = time.Now()
}

func Counter() {
	writer = uilive.New()
	writer.Start()
	time.Sleep(time.Millisecond * 1000) //
	for {
		avgSpeed := total / uint64(time.Since(botStartElapsed).Seconds())
		fmt.Fprintf(writer, "Thread Count = %v\nElapsed Time = %v\nGenerated Wallet = %d\nGenerate Speed Avg(s) = %v\nFound = %d\nFor Close ctrl+c\n", *thread_opt, time.Since(botStartElapsed).String(), total, avgSpeed, totalFound)
		time.Sleep(time.Millisecond * 1000)
	}
	//writer.Stop() // flush and stop rendering
}

func Brute(id int, wg *sync.WaitGroup) {
	//fmt.Println(id)
	defer wg.Done()
	for { ////Elapsed Time 0.0010003
		randomPhrase := RandomPhrase(*phraseCount_opt) //Elapsed Time 0.000999
		//fmt.Println(randomPhrase)
		randomWallet := GeneratorFull(randomPhrase) //Elapsed Time 0.0010002
		//fmt.Println(randomWallet.base58BitcoinAddress)
		if sbf.Test([]byte(randomWallet.base58BitcoinAddress)) {
			SaveWallet(randomWallet)
			//fmt.Println("Bingo:" + randomPhrase)
			totalFound++
		}
		total++
	}
}

// func Brute(id int, wg *sync.WaitGroup) {
// 	defer wg.Done()
// 	for { ////Elapsed Time 0.0010003
// 		randomPhrase := RandomPhrase(*phraseCount_opt) //Elapsed Time 0.000999
// 		Generator(randomPhrase)
// 		// randomWallet := Generator(randomPhrase)                 //Elapsed Time 0.0010002
// 		// if CheckWallet(sqliteDatabase, randomWallet.RIPEM160) { //Elapsed Time 0.0010004
// 		// 	totalFound++
// 		// 	SaveWallet(randomWallet)
// 		// 	fmt.Println("Bingo:" + randomPhrase)
// 		// }
// 		total++
// 	}

// }

func Generator(passphrase string) Wallet {
	hasher := sha256.New() // SHA256
	sha := SHA256(hasher, []byte(passphrase))
	publicKeyBytes := secp256k1.UncompressedPubkeyFromSeckey(sha) // ECDSA
	sha = SHA256(hasher, publicKeyBytes)                          // SHA256
	ripe := RIPEMD160(sha)                                        // RIPEMD160
	versionripeNormal := hex.EncodeToString(ripe)
	return Wallet{RIPEM160: versionripeNormal, passphrase: passphrase} // Send line to output channel
}

func GeneratorFull(passphrase string) Wallet {
	hasher := sha256.New() // SHA256
	sha := SHA256(hasher, []byte(passphrase))

	publicKeyBytes := secp256k1.UncompressedPubkeyFromSeckey(sha) // ECDSA
	privateKey := hex.EncodeToString(sha)                         // Store Private Key

	sha = SHA256(hasher, publicKeyBytes) // SHA256
	ripe := RIPEMD160(sha)               // RIPEMD160

	versionripeNormal := hex.EncodeToString(ripe) // Add version byte 0x00
	versionripe := "00" + versionripeNormal       // Add version byte 0x00
	decoded, _ := hex.DecodeString(versionripe)

	sha = SHA256(hasher, SHA256(hasher, decoded)) // SHA256x2

	addressChecksum := hex.EncodeToString(sha)[0:8] // Concencate Address Checksum and Extended RIPEMD160 Hash
	hexBitcoinAddress := versionripe + addressChecksum

	bigintBitcoinAddress, _ := new(big.Int).SetString((hexBitcoinAddress), 16) // Base58Encode the Address
	base58BitcoinAddress := base58.Encode(bigintBitcoinAddress.Bytes())
	return Wallet{base58BitcoinAddress: "1" + base58BitcoinAddress, RIPEM160: versionripeNormal, privateKey: privateKey, passphrase: passphrase} // Send line to output channel
}

// func Generator(passphrase string) Wallet {
// 	hasher := sha256.New() // SHA256
// 	sha := SHA256(hasher, []byte(passphrase))

// 	publicKeyBytes := secp256k1.UncompressedPubkeyFromSeckey(sha) // ECDSA
// 	privateKey := hex.EncodeToString(sha)                         // Store Private Key

// 	sha = SHA256(hasher, publicKeyBytes) // SHA256
// 	ripe := RIPEMD160(sha)               // RIPEMD160

// 	versionripeNormal := hex.EncodeToString(ripe) // Add version byte 0x00
// 	versionripe := "00" + versionripeNormal       // Add version byte 0x00
// 	decoded, _ := hex.DecodeString(versionripe)

// 	sha = SHA256(hasher, SHA256(hasher, decoded)) // SHA256x2

// 	addressChecksum := hex.EncodeToString(sha)[0:8] // Concencate Address Checksum and Extended RIPEMD160 Hash
// 	hexBitcoinAddress := versionripe + addressChecksum

// 	bigintBitcoinAddress, _ := new(big.Int).SetString((hexBitcoinAddress), 16) // Base58Encode the Address
// 	base58BitcoinAddress := base58.Encode(bigintBitcoinAddress.Bytes())
// 	return Wallet{base58BitcoinAddress: "1" + base58BitcoinAddress, RIPEM160: versionripeNormal, privateKey: privateKey, passphrase: passphrase} // Send line to output channel
// }

func AddressToRIPEM160(address string) string {
	baseBytes := base58.Decode(address)
	end := len(baseBytes) - 4
	hash := baseBytes[0:end]
	return hex.EncodeToString(hash)[2:]
}

func SaveWallet(walletInfo Wallet) {
	fullWallet := GeneratorFull(walletInfo.passphrase)
	f, err := os.OpenFile(*output_opt,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	if _, err := f.WriteString(fullWallet.base58BitcoinAddress + ":" + fullWallet.passphrase + ":" + fullWallet.privateKey + ":" + fullWallet.RIPEM160 + "\n"); err != nil {
		log.Println(err)
	}
}

// func SaveWallet(walletInfo Wallet) {
// 	f, err := os.OpenFile(*output_opt,
// 		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	defer f.Close()
// 	if _, err := f.WriteString(walletInfo.base58BitcoinAddress + ":" + walletInfo.passphrase + ":" + walletInfo.privateKey + ":" + walletInfo.RIPEM160 + "\n"); err != nil {
// 		log.Println(err)
// 	}
// }

// func CheckWallet(db *sql.DB, hash string) bool {
// 	sqlStmt := `SELECT hash FROM hash160 WHERE hash = ?`
// 	err := db.QueryRow(sqlStmt, hash).Scan(&hash)
// 	if err != nil {
// 		if err != sql.ErrNoRows {
// 			log.Print(err)
// 		}
// 		return false
// 	}
// 	return true
// }

// func insertHash160Batch(db *sql.DB, hash string) {
// 	insertHash := `INSERT INTO hash160(hash) VALUES ` + hash
// 	statement, err := db.Prepare(insertHash)

// 	//println(insertHash)
// 	if err != nil {
// 		log.Fatalln(err.Error())
// 	}
// 	_, err = statement.Exec()
// 	if err != nil {
// 		log.Fatalln(err.Error())
// 	}
// }

// func createDB() {
// 	log.Println("Creating Database...")
// 	file, err := os.Create("database.db") // Create SQLite file
// 	if err != nil {
// 		log.Fatal(err.Error())
// 	}
// 	file.Close()
// 	log.Println("DB created")
// }

// func InsertWalletToDB() {
// 	createTable(sqliteDatabase) // Create Database Tables
// 	var tempCodes []string
// 	var currentIndex int = 0
// 	for _, address := range addressList {
// 		tempCodes = append(tempCodes, AddressToRIPEM160(address))
// 		if currentIndex == 10000 {
// 			code := `("` + strings.Join(tempCodes, `") , ("`) + `")`
// 			insertHash160Batch(sqliteDatabase, code)
// 			tempCodes = nil
// 			currentIndex = 0
// 		}
// 		currentIndex++
// 	}
// 	if len(tempCodes) <= 10000 {
// 		insertHash160Batch(sqliteDatabase, `("`+strings.Join(tempCodes, `") , ("`)+`")`)
// 		tempCodes = nil
// 	}
// 	fmt.Println("Wallets Inserted Databased.")
// }

// func createTable(db *sql.DB) {
// 	createHashTable := `CREATE TABLE "hash160" (
// 		"id"	INTEGER UNIQUE,
// 		"hash"	TEXT NOT NULL UNIQUE,
// 		PRIMARY KEY("id" AUTOINCREMENT)
// 	);` // SQL Statement for Create Table
// 	log.Println("Creating table...")
// 	statement, err := db.Prepare(createHashTable) // Prepare SQL Statement
// 	if err != nil {
// 		log.Fatal(err.Error())
// 	}
// 	statement.Exec() // Execute SQL Statements
// 	log.Println("Table created")
// }

// func insertHash160(db *sql.DB, hash string) {
// 	insertHash := `INSERT INTO hash160(hash) VALUES (?)`
// 	statement, err := db.Prepare(insertHash)
// 	if err != nil {
// 		log.Fatalln(err.Error())
// 	}
// 	_, err = statement.Exec(hash)
// 	if err != nil {
// 		log.Fatalln(err.Error())
// 	}
// }

// func displayHashs(db *sql.DB) {
// 	row, err := db.Query("SELECT * FROM hash160")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer row.Close()
// 	for row.Next() {
// 		var hashwallet string
// 		row.Scan(&hashwallet)
// 		//log.Println("Wallet: ", hashwallet)
// 	}
// }

func RandomPhrase(length int) string {
	var phrase []string
	for i := 0; i < length; i++ {
		phrase = append(phrase, wordList[rand.Intn(len(wordList))])
	}
	//lastString := strings.Join(phrase, " ")
	return strings.Join(phrase, " ")
}

// SHA256 Hasher function
func SHA256(hasher hash.Hash, input []byte) (hash []byte) {

	hasher.Reset()
	hasher.Write(input)
	hash = hasher.Sum(nil)
	return hash

}

// RIPEMD160 Hasher function
func RIPEMD160(input []byte) (hash []byte) {

	riper := ripemd160.New()
	riper.Write(input)
	hash = riper.Sum(nil)
	return hash

}
