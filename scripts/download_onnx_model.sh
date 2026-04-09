#!/usr/bin/env bash
set -euo pipefail

MODEL_NAME="paddle_ocr_v2.onnx"
MODEL_DIR="${ONNX_MODEL_DIR:-models}"
MODEL_PATH="${MODEL_DIR}/${MODEL_NAME}"
DOWNLOAD_URL="https://github.com/PaddlePaddle/PaddleOCR/raw/release/2.7/onnx/paddle_ocr_v2.onnx"
EXPECTED_SHA256="e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

mkdir -p "${MODEL_DIR}"

if [ -f "${MODEL_PATH}" ]; then
    ACTUAL=$(sha256sum "${MODEL_PATH}" | awk '{print $1}')
    if [ "${ACTUAL}" = "${EXPECTED_SHA256}" ]; then
        echo "Model already exists and passes SHA256 verification: ${MODEL_PATH}"
        exit 0
    else
        echo "SHA256 mismatch (expected ${EXPECTED_SHA256}, got ${ACTUAL}). Re-downloading..."
        rm -f "${MODEL_PATH}"
    fi
fi

echo "Downloading ${MODEL_NAME} from ${DOWNLOAD_URL}..."
curl -fsSL "${DOWNLOAD_URL}" -o "${MODEL_PATH}"

ACTUAL=$(sha256sum "${MODEL_PATH}" | awk '{print $1}')
if [ "${ACTUAL}" = "${EXPECTED_SHA256}" ]; then
    echo "SHA256 verification passed: ${MODEL_PATH}"
else
    echo "ERROR: SHA256 verification failed!"
    echo "  Expected: ${EXPECTED_SHA256}"
    echo "  Actual:   ${ACTUAL}"
    rm -f "${MODEL_PATH}"
    exit 1
fi

echo "Model downloaded successfully: ${MODEL_PATH}"
echo "Set ONNX_OCR_MODEL_PATH=${MODEL_PATH} to use this model."
