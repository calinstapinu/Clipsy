# from fastapi import FastAPI, HTTPException, Request
# import instaloader
#
# app = FastAPI()
#
# @app.post("/download")
# async def download_instagram(request: Request):
#     try:
#         # Parse JSON body to extract the URL
#         data = await request.json()
#         url = data.get("url")
#
#         if not url:
#             raise HTTPException(status_code=400, detail="URL field is required")
#
#         # Initialize Instaloader
#         loader = instaloader.Instaloader(download_video_thumbnails=False)
#
#         # Extract shortcode from the URL and download the post
#         post = instaloader.Post.from_shortcode(loader.context, url.split("/")[-2])
#         loader.download_post(post, target=f"downloads/{post.owner_username}")
#
#         return {"status": "success", "message": f"Downloaded {post.url}"}
#     except Exception as e:
#         # Return an error message if the download fails
#         raise HTTPException(status_code=400, detail=f"Download failed: {str(e)}")

from fastapi import FastAPI, HTTPException, Request
import instaloader
import os

app = FastAPI()

# Ensure the 'downloads' folder exists
if not os.path.exists("downloads"):
    os.makedirs("downloads")

@app.post("/download")
async def download_instagram(request: Request):
    try:
        # Parse JSON body to extract the URL
        data = await request.json()
        url = data.get("url")

        if not url:
            raise HTTPException(status_code=400, detail="URL field is required")

        # Initialize Instaloader with a specific download directory
        loader = instaloader.Instaloader(download_video_thumbnails=False)
        loader.dirname_pattern = "downloads"  # Set the folder to save all downloads here
        loader.post_metadata_txt_pattern = ""  # Prevents creating metadata files

        # Extract the post shortcode from the URL
        shortcode = url.split("/")[-2]
        post = instaloader.Post.from_shortcode(loader.context, shortcode)

        # Download only if the post has a video
        if post.is_video:
            loader.download_post(post, target="downloads")
        else:
            raise HTTPException(status_code=400, detail="No video found in the specified post")
        # Delete non-MP4 files from the download folder

        for filename in os.listdir("downloads"):
            if not filename.endswith(".mp4"):
                os.remove(os.path.join("downloads", filename))

        return {"status": "success", "message": f"Downloaded video from {url}"}
    except Exception as e:
        # Return an error message if the download fails
        raise HTTPException(status_code=400, detail=f"Download failed: {str(e)}")

