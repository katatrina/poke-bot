#!/usr/bin/env bash
set -e
mkdir -p samples

cat > samples/ai_course_catalog.md <<'MD'
# AI Course Catalog
CyberSoft cung cấp các khoá: Machine Learning cơ bản, Deep Learning nâng cao, Natural Language Processing. Mỗi khoá gồm 8 tuần, bài tập thực hành và project cuối kỳ.
MD

cat > samples/company_overview.md <<'MD'
# Company Overview
CyberSoft là công ty tư vấn đào tạo CNTT, chuyên mảng AI & Cloud. Trụ sở: Hà Nội. Mission: giúp doanh nghiệp áp dụng AI thực dụng.
MD

cat > samples/product_specs.md <<'MD'
# Product Specs - KnowledgeHub
KnowledgeHub v1.0: ingest documents, vector search, web UI. Endpoints: /ingest, /query, /chat. Supported formats: .md, .txt, (pdf/docx later).
MD

cat > samples/hr_policy.md <<'MD'
# HR Policy
Quy định nghỉ phép, đánh giá hiệu suất, chính sách remote. Lưu ý: mọi thông tin lương là bảo mật.
MD

cat > samples/engineering_guidelines.md <<'MD'
# Engineering Guidelines
Coding standards: Go modules, interfaces, context for timeouts, structured logging (zap). PR checklist: tests, lint, docs.
MD

cat > samples/onboarding.md <<'MD'
# Onboarding
Hướng dẫn access repo, run local dev (`make dev`), setup .env (không commit keys), đọc README trước khi bắt tay code.
MD

cat > samples/api_reference.md <<'MD'
# API Reference - RAG Chat Demo
POST /ingest {file}, POST /query {query, top_k}, POST /chat {session_id, query}. Responses include sources with excerpt + score.
MD

cat > samples/research_summary.md <<'MD'
# Research Summary - Retrieval Augmented Generation
RAG combines retrieval from external docs with LLM prompt. Benefits: grounded answers, provenance. Drawbacks: freshness, retrieval quality.
MD

cat > samples/meeting_notes_q1.md <<'MD'
# Meeting Notes Q1
Các quyết định: dùng Qdrant làm vector DB, embeddings OpenAI cho MVP, Alpine.js cho frontend nhẹ.
MD

cat > samples/faq.md <<'MD'
# FAQ
Q: Làm sao để index file mới? A: Gọi /ingest hoặc chạy CLI ingest. Q: Có user account không? A: Không, session-based ephemeral.
MD

cat > samples/tutorial_retrieval.md <<'MD'
# Tutorial - How retrieval works
1. Chunk document → 2. Compute embedding → 3. Upsert to vector DB → 4. Query by embedding → 5. Rerank & assemble prompt.
MD

cat > samples/terms_and_conditions.md <<'MD'
# Terms and Conditions
Dùng demo chỉ để mục đích trình diễn. Không đảm bảo tính chính xác của nội dung. Không upload dữ liệu có quyền sở hữu hay PII.
MD

echo "Created $(ls samples | wc -l) sample files in samples/"
