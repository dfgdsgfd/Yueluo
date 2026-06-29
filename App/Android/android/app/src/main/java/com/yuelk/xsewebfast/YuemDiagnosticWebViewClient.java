package com.yuelk.xsewebfast;

import android.graphics.Bitmap;
import android.net.Uri;
import android.net.http.SslError;
import android.webkit.SslErrorHandler;
import android.webkit.WebResourceError;
import android.webkit.WebResourceRequest;
import android.webkit.WebResourceResponse;
import android.webkit.WebView;
import android.webkit.WebViewClient;

import com.getcapacitor.Bridge;
import com.getcapacitor.BridgeWebViewClient;

final class YuemDiagnosticWebViewClient extends BridgeWebViewClient {
    private final MainActivity activity;
    private final String appUrl;
    private volatile boolean diagnosticsVisible;

    YuemDiagnosticWebViewClient(MainActivity activity, Bridge bridge) {
        super(bridge);
        this.activity = activity;
        this.appUrl = bridge.getAppUrl();
    }

    @Override
    public void onPageStarted(WebView view, String url, Bitmap favicon) {
        if (url != null && appUrl != null && url.startsWith(appUrl)) diagnosticsVisible = false;
        super.onPageStarted(view, url, favicon);
    }

    @Override
    public void onReceivedError(WebView view, WebResourceRequest request, WebResourceError error) {
        if (request != null && request.isForMainFrame() && !diagnosticsVisible) {
            int code = error == null ? 0 : error.getErrorCode();
            String description = error == null || error.getDescription() == null ? "" : error.getDescription().toString();
            showDiagnostics(view, request.getUrl(), webViewErrorText(code, description), null);
            return;
        }
        super.onReceivedError(view, request, error);
    }

    @Override
    public void onReceivedHttpError(WebView view, WebResourceRequest request, WebResourceResponse response) {
        if (request != null && request.isForMainFrame() && !diagnosticsVisible) {
            showDiagnostics(view, request.getUrl(), "主页面返回 HTTP 错误", response == null ? null : response.getStatusCode());
            return;
        }
        super.onReceivedHttpError(view, request, response);
    }

    @Override
    public void onReceivedSslError(WebView view, SslErrorHandler handler, SslError error) {
        Uri target = error == null ? null : Uri.parse(error.getUrl());
        if (!diagnosticsVisible && target != null && isAppUrl(target.toString())) {
            handler.cancel();
            String reason = error == null ? "TLS/SSL 校验失败" : "TLS/SSL 校验失败（代码 " + error.getPrimaryError() + "）";
            showDiagnostics(view, target, reason, null);
            return;
        }
        super.onReceivedSslError(view, handler, error);
    }

    private void showDiagnostics(WebView view, Uri failedUri, String failureReason, Integer httpStatus) {
        diagnosticsVisible = true;
        String target = failedUri == null || failedUri.getScheme() == null ? appUrl : failedUri.toString();
        if (target == null || target.isEmpty()) target = "https://xse.yuelk.com";
        String finalTarget = target;
        view.loadDataWithBaseURL("https://localhost", diagnosticHtml(finalTarget, failureReason, httpStatus), "text/html", "UTF-8", null);

        String userAgent = view.getSettings().getUserAgentString();
        new Thread(() -> {
            YuemNetworkProbe.Result result = YuemNetworkProbe.run(activity.getApplicationContext(), finalTarget, userAgent);
            activity.runOnUiThread(() -> view.evaluateJavascript(
                "window.__yuemDiagnosticUpdate&&window.__yuemDiagnosticUpdate(" + result.toJson().toString() + ")",
                null
            ));
        }, "yuem-network-diagnostics").start();
    }

    private boolean isAppUrl(String url) {
        return url != null && appUrl != null && url.startsWith(appUrl);
    }

    private String diagnosticHtml(String target, String failureReason, Integer httpStatus) {
        String host = "";
        try {
            host = Uri.parse(target).getHost();
        } catch (Exception ignored) {}

        return "<!doctype html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\">"
            + "<meta name=\"viewport\" content=\"width=device-width,initial-scale=1,viewport-fit=cover\">"
            + "<title>月梦网络诊断</title><style>"
            + ":root{color-scheme:dark;font-family:system-ui,-apple-system,sans-serif;background:#0b0f14;color:#e5edf6}"
            + "*{box-sizing:border-box}body{margin:0;min-height:100vh;padding:calc(22px + env(safe-area-inset-top)) 16px calc(22px + env(safe-area-inset-bottom));background:#0b0f14}"
            + "main{max-width:560px;margin:auto}.badge{display:inline-flex;border:1px solid #334155;border-radius:999px;padding:5px 10px;color:#cbd5e1;font-size:12px}"
            + "h1{margin:16px 0 8px;font-size:24px}p{margin:0;color:#94a3b8;line-height:1.55;font-size:14px}"
            + ".panel{margin-top:16px;border:1px solid #233044;background:#111827;border-radius:14px;overflow:hidden}"
            + ".row{display:grid;grid-template-columns:104px minmax(0,1fr);gap:10px;padding:11px 13px;border-top:1px solid #233044;font-size:12px}.row:first-child{border-top:0}"
            + ".label{color:#94a3b8}.value{color:#e2e8f0;white-space:pre-wrap;word-break:break-all;font-family:ui-monospace,SFMono-Regular,Consolas,monospace}"
            + ".ok{color:#34d399}.bad{color:#fb7185}.warn{color:#fbbf24}.actions{display:grid;grid-template-columns:1fr 1fr;gap:10px;margin-top:16px}"
            + ".btn{border:0;border-radius:12px;padding:13px;font:inherit;font-weight:700;text-align:center;text-decoration:none;background:#38bdf8;color:#061018}.secondary{background:#1e293b;color:#e2e8f0;border:1px solid #334155}"
            + ".hint{margin-top:12px;font-size:12px}</style></head><body><main>"
            + "<span class=\"badge\">Yuem Android · 网络诊断</span><h1>无法访问服务器</h1>"
            + "<p>请等待检测完成后直接截图本页。页面不会显示账号、Cookie 或登录令牌。</p><div class=\"panel\">"
            + row("目标地址", escapeHtml(target))
            + row("域名", escapeHtml(host))
            + row("初始错误", escapeHtml(failureReason == null || failureReason.isEmpty() ? "主页面加载失败" : failureReason))
            + row("HTTP 初始码", httpStatus == null ? "无" : String.valueOf(httpStatus))
            + row("网络", valueSpan("network"))
            + row("设备 IP", valueSpan("deviceIps"))
            + row("DNS 服务器", valueSpan("dnsServers"))
            + row("DNS 解析", valueSpan("dns"))
            + row("目标 IP/端口", valueSpan("endpoint"))
            + row("TCP 端口", valueSpan("tcp"))
            + row("HTTPS/HTTP", valueSpan("http"))
            + row("耗时", valueSpan("timing"))
            + row("详细原因", valueSpan("details"))
            + row("WebView UA", valueSpan("ua"))
            + "</div><div class=\"actions\"><a class=\"btn\" href=\"" + escapeHtml(target) + "\">重新连接</a>"
            + "<a class=\"btn secondary\" href=\"" + escapeHtml(target) + "\">重新检测</a></div>"
            + "<p class=\"hint\">判定顺序：设备网络 → DNS → IP:端口 TCP → HTTPS/HTTP。</p>"
            + diagnosticScript() + "</main></body></html>";
    }

    private String diagnosticScript() {
        return "<script>"
            + "function el(id){return document.getElementById(id)}function text(id,v,c){var e=el(id);e.textContent=v||'无';e.className=c||''}"
            + "function list(v){return v&&v.length?v.join('\\n'):'无'}window.__yuemDiagnosticUpdate=function(d){"
            + "var net=d.online?(d.validated?'可联网且已验证':'有网络但未验证'):'离线或无互联网';if(d.captivePortal)net+='（可能需网页认证）';text('network',d.transport+' · '+net,d.validated?'ok':(d.online?'warn':'bad'));"
            + "text('deviceIps',list(d.deviceIps));text('dnsServers',list(d.dnsServers));"
            + "text('dns',d.resolvedIps&&d.resolvedIps.length?list(d.resolvedIps):('失败：'+(d.dnsError||'无解析结果')),d.resolvedIps&&d.resolvedIps.length?'ok':'bad');"
            + "text('endpoint',d.tcpEndpoint||(d.host+':'+d.port));text('tcp',d.tcpReachable?'端口可连接':'端口不可连接：'+(d.tcpError||'未知'),d.tcpReachable?'ok':'bad');"
            + "var hs=d.httpStatus?'状态码 '+d.httpStatus:'无响应：'+(d.httpError||'未知');text('http',hs,d.httpStatus&&d.httpStatus<500?'ok':'bad');"
            + "text('timing','DNS '+d.dnsElapsedMs+' ms · TCP '+d.tcpElapsedMs+' ms · HTTP '+d.httpElapsedMs+' ms');"
            + "text('details',[d.dnsError,d.tcpError,d.httpError,d.fatalError].filter(Boolean).join('\\n')||'未发现额外原生错误');text('ua',d.userAgent||'未知');};"
            + "</script>";
    }

    private String row(String label, String value) {
        return "<div class=\"row\"><div class=\"label\">" + label + "</div><div class=\"value\">" + value + "</div></div>";
    }

    private String valueSpan(String id) {
        return "<span id=\"" + id + "\">检测中…</span>";
    }

    private String webViewErrorText(int code, String description) {
        String category;
        switch (code) {
            case WebViewClient.ERROR_HOST_LOOKUP: category = "DNS 解析失败"; break;
            case WebViewClient.ERROR_CONNECT: category = "服务器或端口连接失败"; break;
            case WebViewClient.ERROR_TIMEOUT: category = "连接或读取超时"; break;
            case WebViewClient.ERROR_FAILED_SSL_HANDSHAKE: category = "TLS/SSL 握手失败"; break;
            case WebViewClient.ERROR_PROXY_AUTHENTICATION: category = "代理服务器认证失败"; break;
            case WebViewClient.ERROR_TOO_MANY_REQUESTS: category = "请求过多"; break;
            default: category = "WebView 加载失败"; break;
        }
        return category + "（" + code + "）" + (description == null || description.isEmpty() ? "" : "：" + description);
    }

    private String escapeHtml(String value) {
        if (value == null) return "";
        return value.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;").replace("\"", "&quot;").replace("'", "&#39;");
    }
}
