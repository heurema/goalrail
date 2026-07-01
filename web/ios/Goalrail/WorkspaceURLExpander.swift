import Foundation

enum WorkspaceURLExpander {
  static func expandIfNeeded(_ url: URL, session: URLSession = .shared) async -> URL {
    _ = session
    return url
  }
}
