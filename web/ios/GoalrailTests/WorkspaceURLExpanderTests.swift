import Foundation
import XCTest

@testable import Goalrail

final class WorkspaceURLExpanderTests: XCTestCase {
  func testReturnsInputURLUnchanged() async {
    let original = URL(string: "https://goalrail.example.com")!
    let expanded = await WorkspaceURLExpander.expandIfNeeded(original)

    XCTAssertEqual(expanded, original)
  }
}
