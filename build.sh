#!/bin/bash

# 构建前端
cd ui
pnpm install
pnpm build
cd ..

# 复制构建产物到后端目录
rm -rf dist
cp -r ui/dist .

# 构建后端
go build -o easyCacheMirror 