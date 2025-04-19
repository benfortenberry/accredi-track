package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	dashboard "github.com/benfortenberry/accredi-track/dashboard"
	employeeLicesnses "github.com/benfortenberry/accredi-track/employeeLicenses"
	employees "github.com/benfortenberry/accredi-track/employees"

	// encoding "github.com/benfortenberry/accredi-track/encoding"
	licenses "github.com/benfortenberry/accredi-track/licenses"
	middleware "github.com/benfortenberry/accredi-track/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var db *sql.DB

func main() {

	envErr := godotenv.Load(".env")
	if envErr != nil {
		fmt.Println("env error")
		log.Fatalf("Error loading .env file: %v", envErr)
	}

	defer db.Close()

	// Capture connection properties.
	cfg := mysql.Config{
		User:                 os.Getenv("DBUSER"),
		Passwd:               os.Getenv("DBPASSWORD"),
		Net:                  "tcp",
		Addr:                 os.Getenv("DBADDR"),
		DBName:               os.Getenv("DBNAME"),
		AllowNativePasswords: true,
	}

	// Get a database handle.
	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")

	// encoding.InitHashids()

	// hd := hashids.NewData()
	// hd.Salt = "your-salt" // Use a strong, unique salt
	// hd.MinLength = 8      // Minimum length of the generated hash
	// h, _ := hashids.NewWithData(hd)

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "https://accreditrack.netlify.app", "http://accreditrack.com"},
		AllowMethods:     []string{"GET, POST, DELETE, PUT"},
		AllowHeaders:     []string{"Content-Type", "Content-Length", "Accept-Encoding", "Authorization", "Cache-Control"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// employee routes
	router.GET("/employees", middleware.AuthMiddleware(), func(c *gin.Context) {
		employees.Get(db, c)
	})

	router.GET("/employee/:id", middleware.AuthMiddleware(), func(c *gin.Context) {
		employees.GetSingle(db, c)
	})

	router.POST("/employees", middleware.AuthMiddleware(), func(c *gin.Context) {
		employees.Post(db, c)
	})
	router.DELETE("/employees/:id", middleware.AuthMiddleware(), func(c *gin.Context) {
		employees.Delete(db, c)
	})
	router.PUT("/employees/:id", middleware.AuthMiddleware(), func(c *gin.Context) {
		employees.Put(db, c)
	})

	// license routes
	router.GET("/licenses", middleware.AuthMiddleware(), func(c *gin.Context) {
		licenses.Get(db, c)
	})

	router.POST("/licenses", middleware.AuthMiddleware(), func(c *gin.Context) {
		licenses.Post(db, c)
	})

	router.PUT("/licenses/:id", middleware.AuthMiddleware(), func(c *gin.Context) {
		licenses.Put(db, c)
	})

	router.DELETE("/licenses/:id", middleware.AuthMiddleware(), func(c *gin.Context) {
		licenses.Delete(db, c)
	})

	// employee license routes
	router.GET("/employee-licenses/:id", middleware.AuthMiddleware(), func(c *gin.Context) {
		employeeLicesnses.Get(db, c)
	})

	router.POST("/employee-licenses", middleware.AuthMiddleware(), func(c *gin.Context) {
		employeeLicesnses.Post(db, c)
	})

	router.PUT("/employee-licenses/:id", middleware.AuthMiddleware(), func(c *gin.Context) {
		employeeLicesnses.Put(db, c)
	})

	router.DELETE("/employee-licenses/:id", middleware.AuthMiddleware(), func(c *gin.Context) {
		employeeLicesnses.Delete(db, c)
	})

	// dashboard routes
	router.GET("/metrics", middleware.AuthMiddleware(), func(c *gin.Context) {
		dashboard.Get(db, c)
	})

	router.GET("/metrics/license-chart-data", middleware.AuthMiddleware(), func(c *gin.Context) {
		dashboard.GetLicenseChartData(db, c)
	})

	router.GET("/metrics/license-chart-data-expired", middleware.AuthMiddleware(), func(c *gin.Context) {
		dashboard.GetExpiredLicenseChartData(db, c)
	})

	router.GET("/metrics/license-chart-data-expiring-soon", middleware.AuthMiddleware(), func(c *gin.Context) {
		dashboard.GetExpiringsByMonth(db, c)
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})

	// Email Notifications
	router.GET("/send-mail", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})
	router.Run(":8080")
}
