"""FastAPI contract tests (no model/GPU): auth gate, input validation,
and fail-closed 503 without ACE-Step installed."""

import os

os.environ.setdefault("GENERATOR_API_TOKEN", "test-token-for-ci-only")

from fastapi.testclient import TestClient

import server

client = TestClient(server.app)
AUTH = {"Authorization": "Bearer test-token-for-ci-only"}


def test_healthz_open_without_token():
    r = client.get("/healthz")
    assert r.status_code == 200
    body = r.json()
    assert body["status"] == "ok"
    assert body["modelLoaded"] is False
    assert body["contractVersions"] == [1, 2]


def test_generate_rejects_missing_and_wrong_token():
    body = {"jobId": "j1", "prompt": "lofi"}
    assert client.post("/generate", json=body).status_code == 401
    assert (
        client.post(
            "/generate", json=body, headers={"Authorization": "Bearer nope"}
        ).status_code
        == 401
    )


def test_generate_no_model_is_503():
    r = client.post(
        "/generate", json={"jobId": "j1", "prompt": "lofi"}, headers=AUTH
    )
    assert r.status_code == 503


def test_generate_validates_before_infer():
    assert (
        client.post(
            "/generate", json={"jobId": "j1", "prompt": "  "}, headers=AUTH
        ).status_code
        == 400
    )
    assert (
        client.post(
            "/generate",
            json={"jobId": "j1", "prompt": "x", "contractVersion": 3},
            headers=AUTH,
        ).status_code
        == 400
    )
    assert (
        client.post("/generate", json={"prompt": "x"}, headers=AUTH).status_code
        == 422
    )
