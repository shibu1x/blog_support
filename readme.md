# Blog Support Project

A Go-based command-line tool for managing Hugo blog posts. It automates the creation of new posts, image processing, and publishing workflows by handling image resizing, S3 uploads, and markdown generation.

## Features

- **Automated Post Creation**: Generate Hugo blog post structure with date-based directories
- **Image Processing**: Automatic resizing (1024x1024) and format conversion (HEIC/WEBP/AVIF → JPG)
- **S3 Integration**: Upload processed images to AWS S3 with automated URL replacement
- **Multiple Posts Per Day**: Support for creating multiple posts on the same date
- **Smart Cleanup**: Remove unused images and local directories after publishing
- **Batch Publishing**: Publish all posts for a specific year in one command

## Prerequisites

- Go 1.24 or higher
- Docker (for containerized deployment)
- ImageMagick (for local development)
- AWS account with S3 bucket (for publishing)

## Installation

### Local Development

1. Clone the repository
2. Copy `.env.sample` to `.env` and configure your settings:
   ```bash
   cp .env.sample .env
   ```
3. Install dependencies:
   ```bash
   go mod download
   ```

### Docker

Build the Docker image using Task:
```bash
task build
```

Or use Docker Compose:
```bash
docker compose up
```

## Configuration

Configure the following environment variables in `.env`:

- `POST_DIR` - Root directory for blog posts (e.g., `content/post`)
- `S3_BUCKET_NAME` - AWS S3 bucket name for image storage
- `REMOTE_IMG_BASE_URL` - Base URL for accessing images
- `AWS_REGION` - AWS region (default: `ap-northeast-1`)
- `S3_KEY_PREFIX` - Prefix for S3 object keys

## Usage

### Create a New Post

Create a post for today:
```bash
go run main.go
```

Create a post for a specific date:
```bash
go run main.go 10/23
go run main.go 2025/10/23
```

Create a second post on the same day:
```bash
go run main.go -N 2 10/23
```

### Publish Posts

Publish all posts for the current year:
```bash
go run main.go -P
```

Publish posts for a specific year:
```bash
go run main.go -P 2025
```

## Workflow

1. Place source images in `{POST_DIR}/YYYY/MM/DD/img_src/`
2. Run the tool to create a post and process images
3. Edit the generated `index.md` with your content
4. Run the publish command to upload images to S3 and finalize

### Directory Structure

```
{POST_DIR}/
  YYYY/
    MM/
      DD/          # Single post per day
        index.md
        img/       # Processed images (deleted after publish)
        img_src/   # Source images (deleted after publish)
      DD_2/        # Second post on same day
        index.md
        img/
        img_src/
```

After publishing, only `index.md` remains with remote image references.

## Image Processing

- **Supported formats**: HEIC, WEBP, AVIF, JPG, JPEG, PNG
- **Output**: JPG (1024x1024), PNG files remain as PNG
- **Filename transformation**: `IMG_1234.heic` → `i1234.jpg`
- **Cover images**: `cover.jpg` or `cover.png` are preserved during cleanup

## License

This project is licensed under the MIT License - see the LICENSE file for details.
