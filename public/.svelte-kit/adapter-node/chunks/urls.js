function appRootDomain(hostname) {
  if (hostname === "localhost" || hostname.endsWith(".localhost") || hostname === "127.0.0.1") {
    return "localhost";
  }
  if (hostname.startsWith("dashboard.")) {
    return hostname.slice("dashboard.".length);
  }
  return hostname;
}
function appScheme(protocol, hostname) {
  const domain = appRootDomain(hostname);
  if (domain === "localhost" || domain === "127.0.0.1") {
    return "http";
  }
  return protocol.replace(":", "") || "https";
}
function projectHost(subdomain, hostname) {
  return `${subdomain}.${appRootDomain(hostname)}`;
}
function projectURL(subdomain, protocol, hostname) {
  return `${appScheme(protocol, hostname)}://${projectHost(subdomain, hostname)}`;
}
export {
  projectHost as a,
  projectURL as p
};
