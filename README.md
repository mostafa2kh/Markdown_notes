## Markdown Notes CLI

A lightweight command-line tool for managing personal notes in Markdown.
Notes are saved as individual JSON files in a local folder, making them easy to back up, search, and export.

## Features

Add notes – Opens your $EDITOR (or vim) to write the body in Markdown.

List notes – Shows all saved notes with IDs, titles, and tags.

View notes – Prints the full note content in the terminal.

Search – Find notes by title, body, or tags (case-insensitive).

Tagging – Add multiple tags to notes for easy organization.

Export – Convert a note into a simple HTML file with minimal Markdown-to-HTML conversion.

## Technologies Used

Language: Go (Golang)

Data Storage: JSON files (notes_db/0001.json, 0002.json, …)


## Project Structure
notes.go        # Main source code
notes_db/       # Database directory (auto-created, stores JSON notes)
README.md       # Documentation

## Installation & Usage
1. Build
go build -o notes notes.go

2. Run Commands
./notes add "Meeting Notes"
./notes list
./notes view 1
./notes search project
./notes tag 1 work research
./notes export 1 note1.html
