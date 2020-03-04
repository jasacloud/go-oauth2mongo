package oauth2mongo

import (
	"context"
	"github.com/jasacloud/go-libraries/db"
	"github.com/jasacloud/go-libraries/db/mongoc"
	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/oauth2.v3"
	"gopkg.in/oauth2.v3/models"
	"log"
)

// ClientConfig client configuration parameters
type ClientConfig struct {
	// store clients data collection name(The default is oauth2_clients)
	ClientsCName string
}

// NewDefaultClientConfig create a default client configuration
func NewDefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		ClientsCName: "oauth2_clients",
	}
}

// NewClientStore create a client store instance based on mongodb
func NewClientStore(conn *mongoc.Connections, ccfgs ...*ClientConfig) *ClientStore {
	err := conn.CheckConnection()
	if err != nil {
		panic(err)
	}

	return NewClientStoreWithSession(conn, ccfgs...)
}

// NewClientStoreWithSession create a client store instance based on mongodb
func NewClientStoreWithSession(conn *mongoc.Connections, ccfgs ...*ClientConfig) *ClientStore {
	cs := &ClientStore{
		conn: conn,
		ccfg: NewDefaultClientConfig(),
	}
	if len(ccfgs) > 0 {
		cs.ccfg = ccfgs[0]
	}

	return cs
}

// ClientStore MongoDB storage for OAuth 2.0
type ClientStore struct {
	ccfg *ClientConfig
	conn *mongoc.Connections
}

// Close close the mongo session
func (cs *ClientStore) Close() {

}

func (cs *ClientStore) c(name string) *mongo.Collection {
	cs.conn.C(name)
	return cs.conn.Collection
}

func (cs *ClientStore) cHandler(name string, handler func(c *mongo.Collection, sc mongo.SessionContext) error) {
	cs.conn.C(name)
	ctx := context.Background()
	var session mongo.Session
	var err error
	if session, err = cs.conn.Client.StartSession(); err != nil {
		log.Println(err)
		return
	}
	if err = session.StartTransaction(); err != nil {
		log.Println(err)
		return
	}
	h := func(sc mongo.SessionContext) error {
		return handler(cs.conn.Collection, sc)
	}
	if err = mongo.WithSession(ctx, session, h); err != nil {
		log.Println(err)
		return
	}
	session.EndSession(ctx)
	return
}

// Set set client information
func (cs *ClientStore) Set(info oauth2.ClientInfo) (err error) {
	cs.cHandler(cs.ccfg.ClientsCName, func(c *mongo.Collection, sc mongo.SessionContext) error {
		entity := &client{
			ID:     info.GetID(),
			Secret: info.GetSecret(),
			Domain: info.GetDomain(),
			UserID: info.GetUserID(),
		}

		if _, cerr := c.InsertOne(context.TODO(), entity); cerr != nil {
			err = cerr
			return err
		}
		return nil
	})

	return
}

// GetByID according to the ID for the client information
func (cs *ClientStore) GetByID(id string) (info oauth2.ClientInfo, err error) {
	cs.cHandler(cs.ccfg.ClientsCName, func(c *mongo.Collection, sc mongo.SessionContext) error {
		entity := new(client)
		if cerr := c.FindOne(context.TODO(), db.Map{"id": id}).Decode(&entity); cerr != nil {
			err = cerr
			return err
		}

		info = &models.Client{
			ID:     entity.ID,
			Secret: entity.Secret,
			Domain: entity.Domain,
			UserID: entity.UserID,
		}
		return nil
	})

	return
}

// RemoveByID use the client id to delete the client information
func (cs *ClientStore) RemoveByID(id string) (err error) {
	cs.cHandler(cs.ccfg.ClientsCName, func(c *mongo.Collection, sc mongo.SessionContext) error {
		if _, cerr := c.DeleteOne(context.TODO(), db.Map{"id": id}); cerr != nil {
			err = cerr
			return err
		}
		return nil
	})

	return
}

type client struct {
	ID     string `bson:"id"`
	Secret string `bson:"secret"`
	Domain string `bson:"domain"`
	UserID string `bson:"userid"`
}
