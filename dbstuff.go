package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Metadata struct {
	PubkeyHex    string `gorm:"primaryKey;size:256"`
	Name         string `gorm:"size:1024"`
	About        string `gorm:"size:4096"`
	Nip05        string `gorm:"size:512"`
	Lud06        string `gorm:"size:2048"`
	Lud16        string `gorm:"size:512"`
	Website      string `gorm:"size:512"`
	DisplayName  string `gorm:"size:512"`
	Picture      string `gorm:"type:text;size:65535"`
	TotalFollows int
	UpdatedAt    time.Time         `gorm:"autoUpdateTime"`
	Follows      []*Metadata       `gorm:"many2many:metadata_follows"`
	Servers      []RecommendServer `gorm:"foreignKey:PubkeyHex;references:PubkeyHex"`
}

type RecommendServer struct {
	PubkeyHex     string    `gorm:"primaryKey;size:256"`
	Url           string    `gorm:"size:512"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
	CreatedAt     time.Time `gorm:"autoUpdateTime"`
	RecommendedBy string    `gorm:"size:256"`
}

type RelayStatus struct {
	Url       string    `gorm:"primaryKey;size:512"`
	Status    string    `gorm:"size:512"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type GormErr struct {
	Number  int    `json:"Number"`
	Message string `json:"Message"`
}

var DB *gorm.DB

func CheckAndReportGormError(err error, allowErrors []string) bool {
	if err != nil {
		for _, e := range allowErrors {
			if err.Error() == e {
				fmt.Println("known error: " + e)
				return true
			}
		}
		byteErr, _ := json.Marshal(err)
		var newError GormErr
		json.Unmarshal((byteErr), &newError)
		fmt.Println(newError)
		return false
	}
	return true
}

func UpdateOrCreateRelayStatus(db *gorm.DB, url string, status string) {
	rowsUpdated := db.Model(RelayStatus{}).Where("url = ?", url).Updates(&RelayStatus{Url: url, Status: status}).RowsAffected
	if rowsUpdated == 0 {
		db.Create(&RelayStatus{Url: url, Status: status})
	}
}

func GetGormConnection() {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,         // Disable color
		},
	)

	dsn := "jeremy:jeremy@tcp(127.0.0.1:3306)/nono?charset=utf8mb4&parseTime=True&loc=UTC"
	db, dberr := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: newLogger})
	if dberr != nil {
		panic("failed to connect database")
	}
	db.Logger.LogMode(logger.Silent)

	DB = db
}
