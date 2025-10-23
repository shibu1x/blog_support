# Blog Support Project

A Go-based CLI tool that automates Hugo blog post creation, image processing, and publishing workflows.

## Features

- **Automated Post Creation**: Generate date-based blog post structure with frontmatter
- **Image Processing**: Resize and convert images (HEIC, WEBP, AVIF → JPG) using ImageMagick
- **S3 Integration**: Upload processed images to AWS S3 with metadata preservation
- **Timestamp Preservation**: Maintains original file timestamps through resize and upload
- **Batch Publishing**: Publish all posts for a given year in one command

## Prerequisites

- Go 1.24 or higher
- Docker (for containerized deployment)
- ImageMagick (for image processing)
- AWS credentials (for S3 uploads)
- Hugo blog with `content/post` directory structure

## Setup

1. Copy the sample environment file and configure:
   ```bash
   cp .env.sample .env
   ```

2. Edit `.env` with your settings:
   ```
   POST_DIR=content/post
   AWS_REGION=ap-northeast-1
   S3_BUCKET_NAME=your-bucket-name
   S3_KEY_PREFIX=your-prefix
   REMOTE_IMG_BASE_URL=https://your-cdn-url
   ```

## Usage

### Create a New Post

```bash
# Create post for today
go run main.go

# Create post for specific date
go run main.go -d 10/23
go run main.go -d 2025/10/23

# Create multiple posts on the same day
go run main.go -d 10/23 -n 2
```

This creates a directory structure with `img/` and `img_src/` folders and an `index.md` file.

### Workflow

1. Place source images in `{POST_DIR}/YYYY/MM/DD/img_src/`
2. Run the tool to process images (resize to 1024x1024, convert formats)
3. Edit the generated `index.md` with your content
4. Run publish command to upload to S3 and finalize

### Publish Posts

```bash
# Publish all posts for current year
go run main.go -p

# Publish posts for specific year
go run main.go -p -y 2025
```

Publishing process:
- Removes unused images (not referenced in `index.md`)
- Uploads images to S3 with original timestamps preserved in metadata
- Replaces local image links with S3 URLs
- Cleans up local `img/` and `img_src/` directories

## Docker Deployment

### Build and Push

Using [Task](https://taskfile.dev/):

```bash
# Build production image
task build

# Build development image
task build_dev
```

### Run with Docker Compose

```bash
docker compose up
```

## Image Processing Details

- **Supported formats**: HEIC, WEBP, AVIF, JPG, JPEG, PNG
- **Output**: 1024x1024 JPG (PNG files remain as PNG)
- **Filename transformation**: `IMG_1234.heic` → `i1234.jpg`
- **Timestamp preservation**: Original file timestamps are preserved through resize and S3 upload

## License

This project is licensed under the MIT License - see the LICENSE file for details.
