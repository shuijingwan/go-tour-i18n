#!/usr/bin/env bash
# 在 repository root 执行；仅 inspection，不 record/export/apply。
go run -mod=readonly ./cmd/tour-i18n quality-check representation-revision \
  --content-package learn-docs-v1 --locale zh-CN \
  --reconciliation-id learn-docs-qc014-machine-successors \
  --previous-snapshot-id learn-docs-v1-qc-014 \
  --previous-snapshot-sha256 234a37330a348590ab3b46e30d720ad42718dc798cb3002e212b86ce47acb619 \
  --formal-impact docs/PROTECTED_REPRESENTATION_FORMAL_IMPACT.json \
  --formal-impact-sha256 8fb332e4ba2b8be55de91c057a3f174ac4bb922536c90e9d4e927e21e6b74e0e \
  --historical-documents docs/protected-representation-evidence/qc014-historical-representations.json \
  --historical-documents-sha256 e119897a3822d9abc42360c46889ac08c850c870f00ad51f250e3b71e91162e4 \
  --id a69694fb9c89d47440249dfd760d315586297f634d199b13532ac95cbced201f \
  --reopen-id learn-docs-qc014-vuln-database-representation-revision
