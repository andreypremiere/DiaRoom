package database

import (
	"context"
	"fmt"
	"encoding/json"
	"time"
	"user-microservice/models"

	"github.com/redis/go-redis/v9"
)

// func AddValue(ctx context.Context, rdb *redis.Client, delay time.Duration, id string, data models.DataForRegister) {
// 	jsonData, err := json.Marshal(data)

// 	if err != nil {
// 		fmt.Println("Ошибка сериализации объекта data models.DataForRegister в AddValue", err.Error())
// 		return
// 	}

// 	executeAt := time.Now().Add(delay).Unix()
// 	newKey := "id:" + id

// 	err = rdb.ZAdd(ctx, newKey, redis.Z{
// 		Score: float64(executeAt),
// 		Member: jsonData,
// 	}).Err()

// 	if err != nil {
// 		fmt.Println("Значение не было добавлено в базу данных", err.Error())
// 		return
// 	}

// }