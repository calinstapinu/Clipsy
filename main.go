package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"github.com/google/go-github/v39/github"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
)

type Video struct {
	ID          int
	Title       string
	Description string
	Genre       string
	UploadDate  string
}

// User struct to represent user details
type User struct {
	ID    int
	Email string
}

var (
	oauth2Config = &oauth2.Config{
		ClientID:     "Ov23li2sU5eoKNPdcAYc",
		ClientSecret: "",
		RedirectURL:  "http://localhost:6969/auth/github/callback",
		Scopes:       []string{"user:email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
	}
	state string
)

// Initialize state with a secure random string
func init() {
	var err error
	state, err = generateRandomString(32) // Generate a secure random string for CSRF protection
	if err != nil {
		log.Fatal("Failed to generate state:", err)
	}
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "videos.db")
	if err != nil {
		return nil, err
	}

	// Create the videos table if it doesn't exist
	createTableSQL := `CREATE TABLE IF NOT EXISTS videos (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT,
        description TEXT,
        genre TEXT,
        upload_date TEXT,
        video BLOB
    );`

	_, err = db.Exec(createTableSQL)
	return db, err
}

func uploadVideoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//log.Println("Handling upload request:", r.Method, r.URL.Path)

		if r.Method != http.MethodPost {
			log.Println("Request method not allowed:", r.Method)
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		log.Println("Handling POST request for file upload")
		err := r.ParseMultipartForm(50 << 20) // 50 MB limit
		if err != nil {
			log.Println("Error parsing form:", err)
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		title := r.FormValue("title")
		description := r.FormValue("description")
		genre := r.FormValue("genre")

		// Log form values to verify they are being received
		log.Printf("Received form values - Title: %s, Description: %s, Genre: %s", title, description, genre)

		// Handle the file upload
		videoFile, _, err := r.FormFile("video")
		if err != nil {
			log.Println("File upload error:", err)
			http.Error(w, "File upload error", http.StatusBadRequest)
			return
		}
		defer videoFile.Close()
		log.Println("File upload successful, reading file data")

		videoData, err := ioutil.ReadAll(videoFile)
		if err != nil {
			log.Println("Error reading file:", err)
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}
		log.Println("Read file data successfully, inserting video into the database")

		// Insert video into the database
		res, err := db.Exec("INSERT INTO videos (title, description, genre, upload_date, video) VALUES (?, ?, ?, datetime('now'), ?)", title, description, genre, videoData)
		if err != nil {
			log.Println("Error saving video:", err)
			http.Error(w, "Error saving video", http.StatusInternalServerError)
			return
		}

		videoID, _ := res.LastInsertId()
		log.Printf("Video inserted successfully with ID: %d", videoID)

		// Prepare a new row for the video table
		video := Video{
			ID:          int(videoID),
			Title:       title,
			Description: description,
			Genre:       genre,
			UploadDate:  "Just now",
		}

		tmpl, err := template.New("videoRow").Parse(`
            <tr>
                <td class="border p-2">{{.Title}}</td>
                <td class="border p-2">{{.Description}}</td>
                <td class="border p-2">{{.Genre}}</td>
                <td class="border p-2">{{.UploadDate}}</td>
                <td class="border p-2">
                    <a href="/download?id={{.ID}}" class="rounded-lg border-solid border-2 border-purple-500 pl-3 pr-3 pb-1 pt-1 text-purple-500 font-semibold transition duration-900 ease-in-out transform hover:bg-purple-500 hover:text-white hover:border-slate-400">Download</a>
                    <form action="/delete" method="post" hx-trigger="submit" hx-encoding="multipart/form-data" hx-target="closest tr" hx-swap="outerHTML" class="inline" enctype="multipart/form-data">
                        <input type="hidden" name="id" value="{{.ID}}">
                        <input type="submit" value="Delete" class="pl-3 pr-3 bg-transparent font-medium cursor-pointer">
                    </form>
                </td>
            </tr>
        `)
		if err != nil {
			log.Println("Template error:", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
			return
		}

		// Return the HTML for the new row
		w.Header().Set("Content-Type", "text/html")
		log.Println("Sending HTML response for the new video row")
		if err := tmpl.Execute(w, video); err != nil {
			log.Println("Error executing template:", err)
			http.Error(w, "Error rendering video row", http.StatusInternalServerError)
		}
	}
}

func deleteVideoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			id := r.FormValue("id")
			deleteSQL := `DELETE FROM videos WHERE id = ?`
			_, err := db.Exec(deleteSQL, id)
			if err != nil {
				log.Println("Error deleting video:", err)
				http.Error(w, "Error deleting video", http.StatusInternalServerError)
				return
			}
			// Return 204 No Content to indicate success
			w.WriteHeader(http.StatusNoContent)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func downloadVideoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing video ID", http.StatusBadRequest)
			return
		}

		var title string
		var videoData []byte
		err := db.QueryRow("SELECT title, video FROM videos WHERE id = ?", id).Scan(&title, &videoData)
		if err != nil {
			log.Println("Video not found:", err)
			http.Error(w, "Video not found", http.StatusNotFound)
			return
		}

		// Set headers for file download
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.mp4", title))
		w.Header().Set("Content-Type", "video/mp4")
		w.Write(videoData)
	}
}

func mainPageHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//log.Println("Serving main page")
		// Query the database for all videos
		rows, err := db.Query("SELECT id, title, description, genre, upload_date FROM videos")
		if err != nil {
			log.Println("Error retrieving videos:", err)
			http.Error(w, "Error retrieving videos", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var videos []Video
		for rows.Next() {
			var v Video
			err := rows.Scan(&v.ID, &v.Title, &v.Description, &v.Genre, &v.UploadDate)
			if err != nil {
				log.Println("Error scanning video:", err)
				http.Error(w, "Error scanning video", http.StatusInternalServerError)
				return
			}
			videos = append(videos, v)
		}

		// Render the main page template with the video list
		tmpl, err := template.ParseFiles("Front/upload.html") // Load from HTML file
		if err != nil {
			log.Println("Error loading template:", err)
			http.Error(w, "Error loading template", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		//log.Println("Executing main page template")
		tmpl.Execute(w, videos)
	}
}

// Auth handler for GitHub login
func authHandler(w http.ResponseWriter, r *http.Request) {
	//log.Println("Client ID:", os.Getenv("GITHUB_CLIENT_ID"))
	//log.Println("Client Secret:", os.Getenv("GITHUB_CLIENT_SECRET"))

	url := oauth2Config.AuthCodeURL(state)
	//log.Println("Redirecting to URL:", url) // Add this log line to debug the URL
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func callbackHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.FormValue("code")
		//log.Println("Received code:", code) // Log the received code

		token, err := oauth2Config.Exchange(r.Context(), code)
		if err != nil {
			http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
			log.Println("Token exchange error:", err)
			return
		}

		log.Println("Token exchanged successfully") // Confirm token exchange

		client := github.NewClient(oauth2Config.Client(r.Context(), token))
		user, _, err := client.Users.Get(r.Context(), "")
		if err != nil {
			log.Println("Failed to get user info:", err) // Log the error
			http.Error(w, "Failed to get user info", http.StatusInternalServerError)
			return
		}

		log.Printf("Logged in user: %s", user.Login)           // Log the logged-in user
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect) // Redirect to the main page
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

//func instagramDownloaderHandler(w http.ResponseWriter, r *http.Request) {
//	tmpl, err := template.ParseFiles("Front/instagram_downloader.html") // Create this HTML file
//	if err != nil {
//		http.Error(w, "Error loading template", http.StatusInternalServerError)
//		return
//	}
//
//	w.Header().Set("Content-Type", "text/html")
//	tmpl.Execute(w, nil)
//}

//func downloadInstagramHandler(w http.ResponseWriter, r *http.Request) {
//	if r.Method == http.MethodPost {
//		url := r.FormValue("url")
//
//	} else {
//		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
//	}
//}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := initDB()
	if err != nil {
		log.Fatal("Failed to initialize the database:", err)
	}
	defer db.Close()

	// Set up routes
	http.HandleFunc("/", mainPageHandler(db))
	http.HandleFunc("/upload", uploadVideoHandler(db))
	http.HandleFunc("/delete", deleteVideoHandler(db))
	http.HandleFunc("/download", downloadVideoHandler(db))
	http.HandleFunc("/auth/github", authHandler)
	http.HandleFunc("/auth/github/callback", callbackHandler(db))
	http.HandleFunc("/logout", logoutHandler)
	//http.HandleFunc("/instagram-downloader", instagramDownloaderHandler)
	//http.HandleFunc("/download-instagram", downloadInstagramHandler)

	http.Handle("/Front/", http.StripPrefix("/Front/", http.FileServer(http.Dir("Front"))))

	fmt.Println("Server is running on http://localhost:6969")
	log.Fatal(http.ListenAndServe(":6969", nil))
}
