package com.yuelk.xsewebfast;

import android.content.Context;
import android.net.ConnectivityManager;
import android.net.LinkAddress;
import android.net.LinkProperties;
import android.net.Network;
import android.net.NetworkCapabilities;

import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

import java.net.HttpURLConnection;
import java.net.InetAddress;
import java.net.InetSocketAddress;
import java.net.Socket;
import java.net.URL;
import java.net.URLConnection;
import java.util.ArrayList;
import java.util.List;

final class YuemNetworkProbe {
    private static final int CONNECT_TIMEOUT_MS = 5000;
    private static final int READ_TIMEOUT_MS = 5000;

    private YuemNetworkProbe() {}

    static Result run(Context context, String targetUrl, String userAgent) {
        Result result = new Result(targetUrl, userAgent);
        ConnectivityManager manager = (ConnectivityManager) context.getSystemService(Context.CONNECTIVITY_SERVICE);
        Network network = manager == null ? null : manager.getActiveNetwork();
        collectNetworkState(manager, network, result);
        try {
            URL url = new URL(targetUrl);
            result.host = url.getHost();
            result.scheme = url.getProtocol();
            result.port = url.getPort() > 0 ? url.getPort() : ("http".equalsIgnoreCase(result.scheme) ? 80 : 443);
            resolve(network, result);
            probeTcp(network, result);
            probeHttp(network, url, result);
        } catch (Exception error) {
            result.fatalError = describe(error);
        }
        return result;
    }

    private static void collectNetworkState(ConnectivityManager manager, Network network, Result result) {
        if (manager == null || network == null) {
            result.transport = "无活动网络";
            return;
        }
        NetworkCapabilities capabilities = manager.getNetworkCapabilities(network);
        result.online = capabilities != null && capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET);
        result.validated = capabilities != null && capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED);
        result.captivePortal = capabilities != null && capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_CAPTIVE_PORTAL);
        result.transport = transportName(capabilities);

        LinkProperties properties = manager.getLinkProperties(network);
        if (properties == null) return;
        for (LinkAddress address : properties.getLinkAddresses()) {
            if (address.getAddress() != null) result.deviceIps.add(address.getAddress().getHostAddress());
        }
        for (InetAddress dns : properties.getDnsServers()) {
            result.dnsServers.add(dns.getHostAddress());
        }
    }

    private static String transportName(NetworkCapabilities capabilities) {
        if (capabilities == null) return "未知";
        if (capabilities.hasTransport(NetworkCapabilities.TRANSPORT_VPN)) return "VPN";
        if (capabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI)) return "Wi-Fi";
        if (capabilities.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR)) return "蜂窝网络";
        if (capabilities.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET)) return "以太网";
        if (capabilities.hasTransport(NetworkCapabilities.TRANSPORT_BLUETOOTH)) return "蓝牙";
        return "其他";
    }

    private static void resolve(Network network, Result result) {
        long start = System.nanoTime();
        try {
            InetAddress[] addresses = network == null ? InetAddress.getAllByName(result.host) : network.getAllByName(result.host);
            for (InetAddress address : addresses) {
                String ip = address.getHostAddress();
                if (ip != null && !result.resolvedIps.contains(ip)) result.resolvedIps.add(ip);
            }
        } catch (Exception error) {
            result.dnsError = describe(error);
        }
        result.dnsElapsedMs = elapsedMs(start);
    }

    private static void probeTcp(Network network, Result result) {
        if (result.resolvedIps.isEmpty()) return;
        InetAddress address;
        try {
            address = InetAddress.getByName(result.resolvedIps.get(0));
        } catch (Exception error) {
            result.tcpError = describe(error);
            return;
        }
        long start = System.nanoTime();
        try (Socket socket = new Socket()) {
            if (network != null) network.bindSocket(socket);
            socket.connect(new InetSocketAddress(address, result.port), CONNECT_TIMEOUT_MS);
            result.tcpReachable = true;
        } catch (Exception error) {
            result.tcpError = describe(error);
        }
        result.tcpEndpoint = displayEndpoint(address.getHostAddress(), result.port);
        result.tcpElapsedMs = elapsedMs(start);
    }

    private static void probeHttp(Network network, URL url, Result result) {
        long start = System.nanoTime();
        HttpURLConnection connection = null;
        try {
            URLConnection raw = network == null ? url.openConnection() : network.openConnection(url);
            connection = (HttpURLConnection) raw;
            connection.setConnectTimeout(CONNECT_TIMEOUT_MS);
            connection.setReadTimeout(READ_TIMEOUT_MS);
            connection.setInstanceFollowRedirects(false);
            connection.setRequestMethod("HEAD");
            connection.setRequestProperty("User-Agent", result.userAgent == null ? "YuemAndroid" : result.userAgent);
            connection.connect();
            result.httpStatus = connection.getResponseCode();
        } catch (Exception error) {
            result.httpError = describe(error);
        } finally {
            result.httpElapsedMs = elapsedMs(start);
            if (connection != null) connection.disconnect();
        }
    }

    private static String displayEndpoint(String ip, int port) {
        return ip != null && ip.contains(":") ? "[" + ip + "]:" + port : ip + ":" + port;
    }

    private static long elapsedMs(long startNanos) {
        return Math.round((System.nanoTime() - startNanos) / 1_000_000.0);
    }

    private static String describe(Exception error) {
        String message = error.getMessage();
        return error.getClass().getSimpleName() + (message == null || message.isEmpty() ? "" : ": " + message);
    }

    static final class Result {
        boolean online;
        boolean validated;
        boolean captivePortal;
        String transport = "未知";
        final String targetUrl;
        String host = "";
        String scheme = "";
        int port;
        final String userAgent;
        final List<String> deviceIps = new ArrayList<>();
        final List<String> dnsServers = new ArrayList<>();
        final List<String> resolvedIps = new ArrayList<>();
        long dnsElapsedMs;
        String dnsError;
        boolean tcpReachable;
        String tcpEndpoint;
        long tcpElapsedMs;
        String tcpError;
        Integer httpStatus;
        long httpElapsedMs;
        String httpError;
        String fatalError;

        Result(String targetUrl, String userAgent) {
            this.targetUrl = targetUrl;
            this.userAgent = userAgent;
        }

        JSONObject toJson() {
            JSONObject json = new JSONObject();
            try {
                json.put("online", online);
                json.put("validated", validated);
                json.put("captivePortal", captivePortal);
                json.put("transport", transport);
                json.put("targetUrl", targetUrl);
                json.put("host", host);
                json.put("scheme", scheme);
                json.put("port", port);
                json.put("userAgent", nullable(userAgent));
                json.put("deviceIps", new JSONArray(deviceIps));
                json.put("dnsServers", new JSONArray(dnsServers));
                json.put("resolvedIps", new JSONArray(resolvedIps));
                json.put("dnsElapsedMs", dnsElapsedMs);
                json.put("dnsError", nullable(dnsError));
                json.put("tcpReachable", tcpReachable);
                json.put("tcpEndpoint", nullable(tcpEndpoint));
                json.put("tcpElapsedMs", tcpElapsedMs);
                json.put("tcpError", nullable(tcpError));
                json.put("httpStatus", nullable(httpStatus));
                json.put("httpElapsedMs", httpElapsedMs);
                json.put("httpError", nullable(httpError));
                json.put("fatalError", nullable(fatalError));
            } catch (JSONException ignored) {
                // Values are limited to JSON primitives and arrays.
            }
            return json;
        }

        private Object nullable(Object value) {
            return value == null ? JSONObject.NULL : value;
        }
    }
}
