#!/bin/bash

# Task Manager Server 起動スクリプト

echo "依存関係をインストール中..."
go get github.com/google/uuid
go mod tidy

echo ""
echo "サーバーを起動します..."
echo ""

go run .

