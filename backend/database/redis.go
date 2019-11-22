package database

import (
	"context"
	"crawlab/entity"
	"crawlab/utils"
	"fmt"
	"github.com/apex/log"
	"github.com/gomodule/redigo/redis"
	"github.com/spf13/viper"
	"runtime/debug"
	"time"
)

var RedisClient *Redis

type Redis struct {
	pool *redis.Pool
}

func NewRedisClient() *Redis {
	return &Redis{pool: NewRedisPool()}
}
func (r *Redis) RPush(collection string, value interface{}) error {
	c := r.pool.Get()
	defer utils.Close(c)

	if _, err := c.Do("RPUSH", collection, value); err != nil {
		debug.PrintStack()
		return err
	}
	return nil
}

func (r *Redis) LPop(collection string) (string, error) {
	c := r.pool.Get()
	defer utils.Close(c)

	value, err2 := redis.String(c.Do("LPOP", collection))
	if err2 != nil {
		return value, err2
	}
	return value, nil
}

func (r *Redis) HSet(collection string, key string, value string) error {
	c := r.pool.Get()
	defer utils.Close(c)

	if _, err := c.Do("HSET", collection, key, value); err != nil {
		debug.PrintStack()
		log.Info(fmt.Sprintf("HSet: collection: %s, key: %s, value: %s", collection, key, value))

		return err
	}
	return nil
}

func (r *Redis) HGet(collection string, key string) (string, error) {
	c := r.pool.Get()
	defer utils.Close(c)

	value, err2 := redis.String(c.Do("HGET", collection, key))
	if err2 != nil {
		return value, err2
	}
	return value, nil
}

func (r *Redis) HDel(collection string, key string) error {
	c := r.pool.Get()
	defer utils.Close(c)

	if _, err := c.Do("HDEL", collection, key); err != nil {
		return err
	}
	return nil
}

func (r *Redis) HKeys(collection string) ([]string, error) {
	c := r.pool.Get()
	defer utils.Close(c)

	value, err2 := redis.Strings(c.Do("HKeys", collection))
	if err2 != nil {
		return []string{}, err2
	}
	return value, nil
}

func NewRedisPool() *redis.Pool {
	var address = viper.GetString("redis.address")
	var port = viper.GetString("redis.port")
	var database = viper.GetString("redis.database")
	var password = viper.GetString("redis.password")

	var url string
	if password == "" {
		url = "redis://" + address + ":" + port + "/" + database
	} else {
		url = "redis://x:" + password + "@" + address + ":" + port + "/" + database
	}
	return &redis.Pool{
		Dial: func() (conn redis.Conn, e error) {
			return redis.DialURL(url,
				redis.DialConnectTimeout(time.Second*10),
				redis.DialReadTimeout(time.Second*10),
				redis.DialWriteTimeout(time.Second*10),
			)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
		MaxIdle:         10,
		MaxActive:       0,
		IdleTimeout:     300 * time.Second,
		Wait:            false,
		MaxConnLifetime: 0,
	}
}

func InitRedis() error {
	RedisClient = NewRedisClient()
	return nil
}

func Pub(channel string, msg entity.NodeMessage) error {
	if _, err := RedisClient.Publish(channel, utils.GetJson(msg)); err != nil {
		log.Errorf("publish redis error: %s", err.Error())
		debug.PrintStack()
		return err
	}
	return nil
}

func Sub(channel string, consume ConsumeFunc) error {
	ctx := context.Background()
	if err := RedisClient.Subscribe(ctx, consume, channel); err != nil {
		log.Errorf("subscribe redis error: %s", err.Error())
		debug.PrintStack()
		return err
	}
	return nil
}
