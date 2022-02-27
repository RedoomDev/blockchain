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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	users := client.Database("blockchain").Collection("users")
	posts := client.Database("blockchain").Collection("posts")
	comments := client.Database("blockchain").Collection("comments")

	blockchain := CreateBlockchain(1)

	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "redoom blockchain system",
		})
	})

	// /new/user
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

		var userCheck bson.M
		err := users.FindOne(context.TODO(), bson.D{
			{"$or", bson.A{
				bson.D{{"email", email}},
				bson.D{{"username", username}},
			}},
		}).Decode(&userCheck)

		if err == nil {
			c.JSON(200, gin.H{
				"error":   true,
				"message": "Bu kullanıcı zaten kayıtlı!",
			})
			return
		}

		doc := bson.D{
			{"username", username},
			{"email", email},
		}

		result, err := users.InsertOne(context.TODO(), doc)

		if err != nil {
			c.JSON(200, gin.H{
				"error":   true,
				"message": "DB de bir hata oldu",
			})
			return
		}

		blockman := blockchain.addBlock(fmt.Sprintf("%v", result.InsertedID.(primitive.ObjectID).Hex()), "users", 1)

		doc2 := bson.D{
			{"username", username},
			{"email", email},
			{"hash", blockman.hash},
			{"blockData", blockman.blockData},
		}

		result2, err := users.ReplaceOne(context.TODO(), bson.D{{"email", email}}, doc2)

		if err != nil {
			c.JSON(200, gin.H{
				"error":   true,
				"message": "DB de bir hata oldu",
			})
			return
		}

		user := map[string]interface{}{
			"username": c.PostForm("username"),
			"email":    c.PostForm("email"),
			"id":       result.InsertedID,
			"result2":  result2,
		}

		c.JSON(200, gin.H{
			"error":     false,
			"message":   "success",
			"hash":      blockman.hash,
			"blockData": blockman.blockData,
			"user":      user,
		})
	})

	// /new/post
	r.POST("/new/post", func(c *gin.Context) {
		user := c.PostForm("id")

		if secret != c.PostForm("secret_key") {
			c.JSON(200, gin.H{
				"error":   true,
				"message": "Bu anahtar yanlış!",
			})
			return
		}

		var userCheck bson.M
		objID, _ := primitive.ObjectIDFromHex(user)
		err := users.FindOne(context.TODO(), bson.D{
			{"_id", objID},
		}).Decode(&userCheck)

		if err != nil {
			c.JSON(200, gin.H{
				"error":   true,
				"message": "Kullanıcı bulunamadı!",
			})
			return
		}

		NewPost := bson.D{
			{"owner", user},
		}

		result, _ := posts.InsertOne(context.TODO(), NewPost)

		blockman := blockchain.addBlock(fmt.Sprintf("%v", result.InsertedID.(primitive.ObjectID).Hex()), user, 1)

		UpdatePost := bson.D{
			{"owner", user},
			{"hash", blockman.hash},
			{"blockData", blockman.blockData},
		}

		update, _ := posts.ReplaceOne(context.TODO(), bson.D{{"_id", result.InsertedID.(primitive.ObjectID)}}, UpdatePost)

		c.JSON(200, gin.H{
			"error":     false,
			"message":   "success",
			"hash":      blockman.hash,
			"blockData": blockman.blockData,
			"post_id":   result.InsertedID,
			"result":    update,
		})
	})

	// /new/comment
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

		ownerID, _ := primitive.ObjectIDFromHex(owner)
		userID, _ := primitive.ObjectIDFromHex(user)

		filter := bson.D{
			{"$or", bson.A{
				bson.D{{"_id", ownerID}},
				bson.D{{"_id", userID}},
			}},
		}

		userCheck, _ := users.Find(context.TODO(), filter)

		var results []bson.M
		if err = userCheck.All(context.TODO(), &results); err != nil {
			log.Fatal(err)
		}

		if len(results) != 2 {
			c.JSON(200, gin.H{
				"error":   true,
				"message": "Kullanıcılar bulunamadı!",
			})
			return
		}

		NewComment := bson.D{
			{"owner", owner},
			{"user_id", user},
			{"post", post},
		}

		result, _ := comments.InsertOne(context.TODO(), NewComment)

		blockman := blockchain.addBlock(fmt.Sprintf("%v", result.InsertedID.(primitive.ObjectID).Hex()), post+"&"+owner+"&"+user, 1)

		UpdateComment := bson.D{
			{"owner", owner},
			{"user_id", user},
			{"post", post},
			{"hash", blockman.hash},
			{"blockData", blockman.blockData},
		}

		update, _ := comments.ReplaceOne(context.TODO(), bson.D{{"_id", result.InsertedID.(primitive.ObjectID)}}, UpdateComment)

		c.JSON(200, gin.H{
			"error":      false,
			"message":    "success",
			"preHash":    blockman.preHash,
			"blockData":  blockman.blockData,
			"comment_id": result.InsertedID,
			"result":     update,
		})
	})
	r.Run(":5000")
}
