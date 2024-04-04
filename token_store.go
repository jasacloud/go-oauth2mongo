package oauth2mongo

import (
	"context"
	"encoding/json"
	"github.com/jasacloud/go-libraries/db"
	"github.com/jasacloud/go-libraries/db/mongoc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"sync"
	"time"

	"gopkg.in/oauth2.v3"
	"gopkg.in/oauth2.v3/models"
)

// TokenConfig token configuration parameters
type TokenConfig struct {
	// store txn collection name(The default is oauth2)
	TxnCName string
	// store token based data collection name(The default is oauth2_basic)
	BasicCName string
	// store access token data collection name(The default is oauth2_access)
	AccessCName string
	// store refresh token data collection name(The default is oauth2_refresh)
	RefreshCName string
}

type StorageIndexed struct {
	sync.RWMutex
	Col map[string]bool
}

var storageIndexed = &StorageIndexed{
	Col: make(map[string]bool),
}

// NewDefaultTokenConfig create a default token configuration
func NewDefaultTokenConfig() *TokenConfig {
	return &TokenConfig{
		TxnCName:     "oauth2_txn",
		BasicCName:   "oauth2_basic",
		AccessCName:  "oauth2_access",
		RefreshCName: "oauth2_refresh",
	}
}

// NewTokenStore create a token store instance based on mongodb
func NewTokenStore(conn *mongoc.Connections, tcfgs ...*TokenConfig) (store *TokenStore) {
	err := conn.CheckConnection()
	if err != nil {
		panic(err)
	}

	return NewTokenStoreWithSession(conn, tcfgs...)
}

// NewTokenStoreWithSession create a token store instance based on mongodb
func NewTokenStoreWithSession(conn *mongoc.Connections, tcfgs ...*TokenConfig) (store *TokenStore) {
	ts := &TokenStore{
		conn: conn,
		tcfg: NewDefaultTokenConfig(),
	}
	if len(tcfgs) > 0 {
		ts.tcfg = tcfgs[0]
	}

	index := mongoc.NewIndex()
	index.AddKeys(bson.D{
		{Key: "ExpiredAt", Value: 1},
	})
	index.CreateIndexesOptions.SetMaxTime(time.Second * 1)

	index2 := mongoc.NewIndex()
	index2.AddKeys(bson.D{
		{Key: "ExpiredAt", Value: 1},
	})
	index2.CreateIndexesOptions.SetMaxTime(time.Second * 1)

	index3 := mongoc.NewIndex()
	index3.AddKeys(bson.D{
		{Key: "ExpiredAt", Value: 1},
	})
	index3.CreateIndexesOptions.SetMaxTime(time.Second * 1)

	storageIndexed.RLock()
	if storageIndexed.Col[conn.Collection.Name()] {
		storageIndexed.RUnlock()
		return
	}
	storageIndexed.RUnlock()

	_, err := conn.CreateIndexes(index, index2, index3)
	if err != nil {
		log.Printf("NewTokenStoreWithSession->CreateIndexes to '%s' returned error: %v \n", conn.Collection.Name(), err)
	}
	storageIndexed.Lock()
	storageIndexed.Col[conn.Collection.Name()] = true
	storageIndexed.Unlock()

	store = ts
	return
}

// TokenStore MongoDB storage for OAuth 2.0
type TokenStore struct {
	tcfg *TokenConfig
	conn *mongoc.Connections
}

// Close close the mongo session
func (ts *TokenStore) Close() {
	//ts.session.Close()
}

func (ts *TokenStore) c(name string) *mongo.Collection {
	ts.conn.C(name)
	return ts.conn.Collection
}

func (ts *TokenStore) cHandler(name string, handler func(c *mongo.Collection)) {
	ts.conn.C(name)
	handler(ts.conn.Collection)
	return
}

// Create create and store the new token information
func (ts *TokenStore) Create(info oauth2.TokenInfo) (err error) {
	jv, err := json.Marshal(info)
	if err != nil {
		return
	}

	if code := info.GetCode(); code != "" {
		ts.cHandler(ts.tcfg.BasicCName, func(c *mongo.Collection) {
			_, err = c.InsertOne(context.TODO(), basicData{
				ID:        code,
				Data:      jv,
				ExpiredAt: info.GetCodeCreateAt().Add(info.GetCodeExpiresIn()),
			})
		})
		return
	}

	aexp := info.GetAccessCreateAt().Add(info.GetAccessExpiresIn())
	rexp := aexp
	if refresh := info.GetRefresh(); refresh != "" {
		rexp = info.GetRefreshCreateAt().Add(info.GetRefreshExpiresIn())
		if aexp.Second() > rexp.Second() {
			aexp = rexp
		}
	}
	id := primitive.NewObjectID().String()
	ts.cHandler(ts.tcfg.BasicCName, func(c *mongo.Collection) {
		_, err = c.InsertOne(context.TODO(), basicData{
			ID:        id,
			Data:      jv,
			ExpiredAt: rexp,
		})
	})

	ts.cHandler(ts.tcfg.AccessCName, func(c *mongo.Collection) {
		_, err = c.InsertOne(context.TODO(), tokenData{
			ID:        info.GetAccess(),
			BasicID:   id,
			ExpiredAt: aexp,
		})
	})

	if refresh := info.GetRefresh(); refresh != "" {
		ts.cHandler(ts.tcfg.RefreshCName, func(c *mongo.Collection) {
			_, err = c.InsertOne(context.TODO(), tokenData{
				ID:        refresh,
				BasicID:   id,
				ExpiredAt: rexp,
			})
		})
	}

	return
}

// RemoveByCode use the authorization code to delete the token information
func (ts *TokenStore) RemoveByCode(code string) (err error) {
	ts.cHandler(ts.tcfg.BasicCName, func(c *mongo.Collection) {
		if _, verr := c.DeleteOne(context.TODO(), db.Map{"id": code}); verr != nil {
			if verr == mongo.ErrNoDocuments {
				return
			}
			err = verr
		}
	})
	return
}

// RemoveByAccess use the access token to delete the token information
func (ts *TokenStore) RemoveByAccess(access string) (err error) {
	ts.cHandler(ts.tcfg.AccessCName, func(c *mongo.Collection) {
		if _, verr := c.DeleteOne(context.TODO(), db.Map{"id": access}); verr != nil {
			if verr == mongo.ErrNoDocuments {
				return
			}
			err = verr
		}
	})
	return
}

// RemoveByRefresh use the refresh token to delete the token information
func (ts *TokenStore) RemoveByRefresh(refresh string) (err error) {
	ts.cHandler(ts.tcfg.RefreshCName, func(c *mongo.Collection) {
		if _, verr := c.DeleteOne(context.TODO(), db.Map{"id": refresh}); verr != nil {
			if verr == mongo.ErrNoDocuments {
				return
			}
			err = verr
		}
	})
	return
}

func (ts *TokenStore) getData(basicID string) (ti oauth2.TokenInfo, err error) {
	ts.cHandler(ts.tcfg.BasicCName, func(c *mongo.Collection) {
		var bd basicData
		if verr := c.FindOne(context.TODO(), db.Map{"id": basicID}).Decode(&bd); verr != nil {
			if verr == mongo.ErrNoDocuments {
				return
			}
			err = verr
		}
		var tm models.Token
		err = json.Unmarshal(bd.Data, &tm)
		if err != nil {
			return
		}
		ti = &tm
	})
	return
}

func (ts *TokenStore) getBasicID(cname, token string) (basicID string, err error) {
	ts.cHandler(cname, func(c *mongo.Collection) {
		var td tokenData
		if verr := c.FindOne(context.TODO(), db.Map{"id": token}).Decode(&td); verr != nil {
			if verr == mongo.ErrNoDocuments {
				return
			}
			err = verr
		}
		basicID = td.BasicID
	})
	return
}

// GetByCode use the authorization code for token information data
func (ts *TokenStore) GetByCode(code string) (ti oauth2.TokenInfo, err error) {
	ti, err = ts.getData(code)
	return
}

// GetByAccess use the access token for token information data
func (ts *TokenStore) GetByAccess(access string) (ti oauth2.TokenInfo, err error) {
	basicID, err := ts.getBasicID(ts.tcfg.AccessCName, access)
	if err != nil && basicID == "" {
		return
	}
	ti, err = ts.getData(basicID)
	return
}

// GetByRefresh use the refresh token for token information data
func (ts *TokenStore) GetByRefresh(refresh string) (ti oauth2.TokenInfo, err error) {
	basicID, err := ts.getBasicID(ts.tcfg.RefreshCName, refresh)
	if err != nil && basicID == "" {
		return
	}
	ti, err = ts.getData(basicID)
	return
}

type basicData struct {
	ID        string    `bson:"id"`
	Data      []byte    `bson:"Data"`
	ExpiredAt time.Time `bson:"ExpiredAt"`
}

type tokenData struct {
	ID        string    `bson:"id"`
	BasicID   string    `bson:"BasicID"`
	ExpiredAt time.Time `bson:"ExpiredAt"`
}
