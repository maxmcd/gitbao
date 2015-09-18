package model

import (
	"log"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Bao struct {
	ID           bson.ObjectId `bson:"_id"`
	GistId       string        `bson:"gist_id"`
	FunctionName string        `bson:"function_name"`
	Ts           time.Time     `bson:"ts"`
}

var bc *mgo.Collection

func init() {
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	bc = session.DB("gitbao").C("bao")
}

func CreateBao(gistId, functionName string) (id bson.ObjectId, err error) {
	id = bson.NewObjectId()
	bao := Bao{
		Ts: time.Now(),
		ID: id,
	}
	err = bc.Insert(bao)
	return
}

func GetAllBaos() (baos []Bao) {
	err := bc.Find(nil).All(&baos)
	log.Fatal(err)
	return
}
