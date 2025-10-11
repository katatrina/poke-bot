# Các bước Refactor Frontend → Vite + Embed

## 1. Setup Vite Project

```bash
cd web
npm create vite@latest . -- --template react-ts
npm install
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

## 2. Tạo cấu trúc thư mục

```bash
mkdir -p src/{components,hooks,services,types}
```

## 3. Copy/Create các files

- `vite.config.ts` - Từ artifact
- `tailwind.config.js` - Từ artifact
- `tsconfig.json` - Từ artifact
- `package.json` - Update scripts
- `index.html` - Từ artifact
- `src/index.css` - Từ artifact
- `src/main.tsx` - Từ artifact
- `src/types/index.ts` - Từ artifact
- `src/hooks/useLocalStorage.ts` - Từ artifact
- `src/services/api.ts` - Từ artifact
- `src/components/*.tsx` - Tất cả components từ artifact
- `src/App.tsx` - Từ artifact

## 4. Install dependencies

```bash
cd web
npm install
```

## 5. Test dev server

```bash
npm run dev
# Mở http://localhost:5173
```

## 6. Build frontend

```bash
npm run build
# Output: web/dist/
```

## 7. Update Go backend

- Update `internal/server/server.go` với embed code
- Thêm `//go:embed all:../web/dist` ở đầu file

## 8. Update go.mod (nếu cần)

```bash
go mod tidy
```

## 9. Test production build

```bash
# Build frontend
cd web && npm run build

# Build backend
cd ../..
go build -o bin/pokebot .

# Run
./bin/pokebot
# Truy cập http://localhost:8080
```

## 10. Update .gitignore

```
web/node_modules/
web/dist/
web/dist/
*.log
```

## 11. Xóa file cũ

```bash
rm web/index.html  # File HTML cũ không dùng nữa
```

## 12. Update Makefile (optional)

```makefile
dev-frontend:
	cd web && npm run dev

build-frontend:
	cd web && npm run build

build: build-frontend
	go build -o bin/pokebot .
```

**Xong!**

Development: `make dev` (2 terminals)  
Production: `make build && ./bin/pokebot`