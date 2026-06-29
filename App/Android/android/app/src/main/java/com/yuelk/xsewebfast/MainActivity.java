package com.yuelk.xsewebfast;

import android.os.Build;
import android.webkit.CookieManager;
import android.webkit.WebSettings;
import android.webkit.WebView;

import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {
    @Override
    protected void load() {
        super.load();
        if (bridge == null) return;

        WebView webView = bridge.getWebView();
        configureWebView(webView);
        bridge.setWebViewClient(new YuemDiagnosticWebViewClient(this, bridge));
        webView.reload();
    }

    private void configureWebView(WebView webView) {
        WebSettings settings = webView.getSettings();
        settings.setJavaScriptEnabled(true);           // Next.js 强烈依赖 JS
        settings.setDomStorageEnabled(true);           // localStorage / sessionStorage
        settings.setDatabaseEnabled(true);             // 部分 IndexedDB 支持

        settings.setCacheMode(WebSettings.LOAD_DEFAULT);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            settings.setOffscreenPreRaster(true);
        }
        CookieManager cookieManager = CookieManager.getInstance();
        cookieManager.setAcceptCookie(true);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
            cookieManager.setAcceptThirdPartyCookies(webView, true);
        }
    }
}
