package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

type Video struct {
	ID          int
	Title       string
	Description string
	Genre       string
	UploadDate  string
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
		log.Println("Handling upload request:", r.Method, r.URL.Path)

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
		log.Println("Serving main page")
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
		log.Println("Executing main page template")
		tmpl.Execute(w, videos)
	}
}

func main() {
	db, err := initDB()
	if err != nil {
		log.Fatal("Failed to initialize the database:", err)
	}
	defer db.Close()

	http.HandleFunc("/", mainPageHandler(db)) // Serve the main page with the video list
	http.HandleFunc("/upload", uploadVideoHandler(db))
	http.HandleFunc("/delete", deleteVideoHandler(db))
	http.HandleFunc("/download", downloadVideoHandler(db))

	// Serve static files from the Front directory
	http.Handle("/Front/", http.StripPrefix("/Front/", http.FileServer(http.Dir("Front"))))

	fmt.Println("Server is running on http://localhost:6969")
	log.Fatal(http.ListenAndServe(":6969", nil))
}
