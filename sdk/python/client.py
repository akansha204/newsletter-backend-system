from __future__ import annotations

import json
import os
from dataclasses import dataclass, field
from typing import Any, Dict, Mapping, Optional
from urllib import error, parse, request

ENV_BASE_URL = "NEWSLETTER_BASE_URL"
ENV_API_KEY = "NEWSLETTER_API_KEY"
API_PREFIX = "/api/v1"


class APIError(Exception):
    def __init__(self, status_code: int, message: str, body: str = "", data: Any = None) -> None:
        super().__init__(f"newsletter sdk request failed with status {status_code}: {message}")
        self.status_code = status_code
        self.message = message
        self.body = body
        self.data = data


@dataclass
class MessageResponse:
    message: str = ""
    error: str = ""

    @classmethod
    def from_payload(cls, payload: Mapping[str, Any]) -> "MessageResponse":
        return cls(
            message=str(payload.get("message", "")),
            error=str(payload.get("error", "")),
        )


@dataclass
class NewsletterSendResponse:
    message: str
    total: int

    @classmethod
    def from_payload(cls, payload: Mapping[str, Any]) -> "NewsletterSendResponse":
        return cls(
            message=str(payload.get("message", "")),
            total=int(payload.get("total", 0)),
        )


@dataclass
class HealthCheck:
    status: str
    error: str = ""

    @classmethod
    def from_payload(cls, payload: Mapping[str, Any]) -> "HealthCheck":
        return cls(
            status=str(payload.get("status", "")),
            error=str(payload.get("error", "")),
        )


@dataclass
class HealthResponse:
    status: str
    checks: Dict[str, HealthCheck] = field(default_factory=dict)

    @classmethod
    def from_payload(cls, payload: Mapping[str, Any]) -> "HealthResponse":
        raw_checks = payload.get("checks", {})
        checks: Dict[str, HealthCheck] = {}
        if isinstance(raw_checks, Mapping):
            for name, check in raw_checks.items():
                if isinstance(check, Mapping):
                    checks[str(name)] = HealthCheck.from_payload(check)

        return cls(
            status=str(payload.get("status", "")),
            checks=checks,
        )


class NewsletterClient:
    def __init__(
        self,
        base_url: Optional[str] = None,
        api_key: Optional[str] = None,
        timeout: float = 10.0,
        default_headers: Optional[Mapping[str, str]] = None,
    ) -> None:
        self._base_url = self._resolve_base_url(base_url)
        self._api_key = (api_key or os.getenv(ENV_API_KEY) or "").strip() or None
        self._timeout = timeout
        self._default_headers = dict(default_headers or {})

    @classmethod
    def from_env(
        cls,
        *,
        timeout: float = 10.0,
        default_headers: Optional[Mapping[str, str]] = None,
    ) -> "NewsletterClient":
        return cls(
            timeout=timeout,
            default_headers=default_headers,
        )

    def set_api_key(self, api_key: str) -> None:
        self._api_key = api_key

    def subscribe(self, email: str) -> MessageResponse:
        if not email or not email.strip():
            raise ValueError("email is required")

        payload = self._request_json(
            "POST",
            "/api/v1/subscribe",
            body={"email": email},
        )
        return MessageResponse.from_payload(payload)

    def confirm(self, token: str) -> MessageResponse:
        if not token or not token.strip():
            raise ValueError("token is required")

        payload = self._request_json(
            "GET",
            "/api/v1/confirm",
            query={"token": token},
        )
        return MessageResponse.from_payload(payload)

    def send_newsletter(
        self,
        subject: str,
        body: str,
        *,
        idempotency_key: Optional[str] = None,
        api_key: Optional[str] = None,
    ) -> NewsletterSendResponse:
        if not subject or not subject.strip():
            raise ValueError("subject is required")
        if not body or not body.strip():
            raise ValueError("body is required")

        resolved_api_key = api_key or self._api_key
        if not resolved_api_key or not resolved_api_key.strip():
            raise ValueError("api_key is required for newsletter sends")

        headers = {"X-API-Key": resolved_api_key}
        if idempotency_key:
            headers["Idempotency-Key"] = idempotency_key

        payload = self._request_json(
            "POST",
            "/api/v1/newsletter/send",
            body={"subject": subject, "body": body},
            headers=headers,
        )
        return NewsletterSendResponse.from_payload(payload)

    def health(self) -> HealthResponse:
        payload = self._request_json("GET", "/api/v1/health")
        return HealthResponse.from_payload(payload)

    def metrics(self) -> str:
        return self._request_text("GET", "/api/v1/metrics")

    def _request_json(
        self,
        method: str,
        path: str,
        *,
        body: Optional[Mapping[str, Any]] = None,
        headers: Optional[Mapping[str, str]] = None,
        query: Optional[Mapping[str, str]] = None,
    ) -> Dict[str, Any]:
        raw = self._request(
            method,
            path,
            body=body,
            headers=headers,
            query=query,
        )
        if not raw:
            return {}

        payload = json.loads(raw)
        if not isinstance(payload, dict):
            raise ValueError("expected a JSON object response from the API")

        return payload

    def _request_text(
        self,
        method: str,
        path: str,
        *,
        headers: Optional[Mapping[str, str]] = None,
        query: Optional[Mapping[str, str]] = None,
    ) -> str:
        return self._request(method, path, headers=headers, query=query)

    def _request(
        self,
        method: str,
        path: str,
        *,
        body: Optional[Mapping[str, Any]] = None,
        headers: Optional[Mapping[str, str]] = None,
        query: Optional[Mapping[str, str]] = None,
    ) -> str:
        url = self._build_url(path, query)
        merged_headers = dict(self._default_headers)
        if headers:
            merged_headers.update(headers)

        data = None
        if body is not None:
            data = json.dumps(body).encode("utf-8")
            merged_headers["Content-Type"] = "application/json"

        req = request.Request(url=url, data=data, headers=merged_headers, method=method)

        try:
            with request.urlopen(req, timeout=self._timeout) as response:
                return response.read().decode("utf-8")
        except error.HTTPError as exc:
            raw_body = exc.read().decode("utf-8")
            raise self._build_api_error(exc.code, raw_body) from exc
        except error.URLError as exc:
            raise RuntimeError(f"newsletter sdk request failed: {exc.reason}") from exc

    def _build_url(self, path: str, query: Optional[Mapping[str, str]]) -> str:
        url = f"{self._base_url}{path}"
        if query:
            return f"{url}?{parse.urlencode(query)}"
        return url

    @staticmethod
    def _build_api_error(status_code: int, raw_body: str) -> APIError:
        message = f"request failed with status {status_code}"
        data: Any = None

        if raw_body.strip():
            try:
                data = json.loads(raw_body)
            except json.JSONDecodeError:
                message = raw_body.strip()
            else:
                if isinstance(data, Mapping):
                    message = str(data.get("error") or data.get("message") or message)

        return APIError(status_code=status_code, message=message, body=raw_body, data=data)

    @staticmethod
    def _resolve_base_url(base_url: Optional[str]) -> str:
        resolved = (base_url or os.getenv(ENV_BASE_URL) or "").strip().rstrip("/")
        if not resolved:
            raise ValueError("base_url is required or set NEWSLETTER_BASE_URL")

        if resolved.endswith(API_PREFIX):
            resolved = resolved[: -len(API_PREFIX)].rstrip("/")

        parsed = parse.urlparse(resolved)
        if not parsed.scheme or not parsed.netloc:
            raise ValueError("base_url must be an absolute URL")

        return resolved