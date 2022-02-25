package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Block struct {
	data         map[string]interface{}
	hash         string
	previousHash string
	timestamp    time.Time
	pow          int
}

type Blockchain struct {
	genesisBlock Block
	chain        []Block
	difficulty   int
}

type Data struct {
	preHash   string
	hash      string
	blockData map[string]interface{}
}

func (b Block) calculateHash() string {
	data, _ := json.Marshal(b.data)
	blockData := b.previousHash + string(data) + b.timestamp.String() + strconv.Itoa(b.pow)
	blockHash := sha256.Sum256([]byte(blockData))
	return fmt.Sprintf("%x", blockHash)
}

func (b *Block) mine(difficulty int) {
	for !strings.HasPrefix(b.hash, strings.Repeat("0", difficulty)) {
		b.pow++
		b.hash = b.calculateHash()
	}
}

func CreateBlockchain(difficulty int) Blockchain {
	genesisBlock := Block{
		hash:      "0",
		timestamp: time.Now(),
	}
	return Blockchain{
		genesisBlock,
		[]Block{genesisBlock},
		difficulty,
	}
}

func (b *Blockchain) addBlock(from, to string, amount float64) Data {
	blockData := map[string]interface{}{
		"from":   from,
		"to":     to,
		"amount": amount,
	}
	lastBlock := b.chain[len(b.chain)-1]
	newBlock := Block{
		data:         blockData,
		previousHash: lastBlock.hash,
		timestamp:    time.Now(),
	}
	newBlock.mine(b.difficulty)
	b.chain = append(b.chain, newBlock)
	data := Data{
		preHash:   newBlock.previousHash,
		hash:      b.chain[len(b.chain)-1].hash,
		blockData: blockData,
	}
	return data
}

func (b Blockchain) isValid() bool {
	for i := range b.chain[1:] {
		previousBlock := b.chain[i]
		currentBlock := b.chain[i+1]
		if currentBlock.hash != currentBlock.calculateHash() || currentBlock.previousHash != previousBlock.hash {
			return false
		}
	}
	return true
}

func main() {

	godotenv.Load(".env")
	uri := os.Getenv("MONGO_URI")
	secret := os.Getenv("SECRET_KEY")

	if uri == "" {
		log.Fatal("You must set your 'MONGO_URI' environmental variable. See\n\t https://docs.mongodb.com/drivers/go/current/usage-examples/#environment-variable")
	}
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	blockchain := CreateBlockchain(1)

	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "redoom blockchain system",
		})
	})

	r.POST("/new/user", func(c *gin.Context) {

		if secret != c.PostForm("secret_key") {
			c.JSON(200, gin.H{
				"error":   true,
				"message": "Bu anahtar yanlış!",
			})
			return
		}

		username := c.PostForm("username")
		email := c.PostForm("email")
		if username == "" || email == "" {
			c.JSON(200, gin.H{
				"error":   true,
				"message": "Hepsini doldurun",
			})
			return
		}

		user := map[string]interface{}{
			"username": c.PostForm("username"),
			"email":    c.PostForm("email"),
		}

		fmt.Print(user)
		blockman := blockchain.addBlock(c.PostForm("id"), "users", 1)

		c.JSON(200, gin.H{
			"error":     false,
			"message":   "success",
			"hash":      blockman.hash,
			"preHash":   blockman.preHash,
			"blockData": blockman.blockData,
		})
	})
	r.POST("/new/post", func(c *gin.Context) {
		user := c.PostForm("id")

		if secret != c.PostForm("secret_key") {
			c.JSON(200, gin.H{
				"error":   true,
				"message": "Bu anahtar yanlış!",
			})
			return
		}

		blockman := blockchain.addBlock(c.PostForm("post_id"), user, 1)

		c.JSON(200, gin.H{
			"error":     false,
			"message":   "success",
			"hash":      blockman.hash,
			"preHash":   blockman.preHash,
			"blockData": blockman.blockData,
		})
	})
	r.POST("/new/comment", func(c *gin.Context) {

		user := c.PostForm("id")
		owner := c.PostForm("post_owner_id")
		post := c.PostForm("post_id")

		if secret != c.PostForm("secret_key") {
			c.JSON(200, gin.H{
				"error":   true,
				"message": "Bu anahtar yanlış!",
			})
			return
		}

		blockman := blockchain.addBlock(c.PostForm("comment_id"), post+"&"+owner+"&"+user, 1)

		c.JSON(200, gin.H{
			"error":     false,
			"message":   "success",
			"hash":      blockman.hash,
			"preHash":   blockman.preHash,
			"blockData": blockman.blockData,
		})
	})
	r.Run(":5000")
}
