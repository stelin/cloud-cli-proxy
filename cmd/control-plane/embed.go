package main

import "embed"

//go:embed dist/*
var adminDist embed.FS
