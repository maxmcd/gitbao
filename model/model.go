package model

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Bao struct {
	ID           bson.ObjectId `bson:"_id"`
	GistId       string        `bson:"gist_id"`
	FunctionName string        `bson:"function_name"`
	Secret       string        `bson:"string"`
	Ts           time.Time     `bson:"ts"`
}

var bc *mgo.Collection

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func init() {
	rand.Seed(time.Now().UnixNano())

	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	bc = session.DB("gitbao").C("bao")
}

func createSecret() string {
	length := 20
	b := make([]byte, length)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func CreateBao(gistId, functionName string) (bao Bao, err error) {
	id := bson.NewObjectId()
	bao = Bao{
		Ts:           time.Now(),
		ID:           id,
		FunctionName: functionName,
		GistId:       gistId,
		Secret:       createSecret(),
	}
	err = bc.Insert(bao)
	return
}

func ConfirmSecret(id, secret string) (isValid bool, err error) {
	bao, err := GetBaoById(id)
	return (bao.Secret == secret), err
}

func GetAllBaos() (baos []Bao) {
	err := bc.Find(nil).All(&baos)
	if err != nil {
		log.Fatal(err)
	}
	return
}

func GetBaoById(id string) (bao Bao, err error) {
	isId := bson.IsObjectIdHex(id)
	if isId == true {
		objId := bson.ObjectIdHex(id)
		err = bc.Find(bson.M{"_id": objId}).One(&bao)
		return
	}
	err = fmt.Errorf("Mongo hex id not valid")
	return
}
