#!/usr/bin/env python3
"""
Test script for Phase 2 Web Skills (Day 1)
Tests the new web.search and web.extract skills.
"""

import asyncio
import sys
import os

# Add the container/openclaw directory to Python path to import bridge_client
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), 'container', 'openclaw'))

from bridge_client import AsyncBridgeClient


async def test_web_search():
    """Test the web.search skill."""
    print("=== Testing Web Search Skill ===")
    
    client = AsyncBridgeClient()
    
    try:
        # Test basic web search
        print("Testing DuckDuckGo search for 'ArmorClaw'...")
        result = await client.skills_web_search(
            query="ArmorClaw",
            engine="duckduckgo",
            per_page=5
        )
        
        print(f"Search Result:")
        print(f"  Query: {result.get('query', 'N/A')}")
        print(f"  Engine: {result.get('engine', 'N/A')}")
        print(f"  Results count: {len(result.get('results', []))}")
        print(f"  Search time: {result.get('search_time', 'N/A')}")
        
        if result.get('results'):
            first_result = result['results'][0]
            print(f"  First result:")
            print(f"    Title: {first_result.get('title', 'N/A')}")
            print(f"    URL: {first_result.get('url', 'N/A')}")
            print(f"    Snippet: {first_result.get('snippet', 'N/A')[:100]}...")
        
        print("✅ Web search test passed\n")
        return True
        
    except Exception as e:
        print(f"❌ Web search test failed: {e}\n")
        return False


async def test_web_extract():
    """Test the web.extract skill."""
    print("=== Testing Web Extract Skill ===")
    
    client = AsyncBridgeClient()
    
    try:
        # Test web extraction with example.com
        print("Testing web extraction from example.com...")
        result = await client.skills_web_extract(
            url="http://example.com",
            content_type="html",
            max_length=5000,
            include_links=True,
            include_images=False,
            include_tables=False
        )
        
        print(f"Extraction Result:")
        print(f"  URL: {result.get('url', 'N/A')}")
        print(f"  Title: {result.get('title', 'N/A')}")
        print(f"  Content type: {result.get('content_type', 'N/A')}")
        print(f"  Word count: {result.get('word_count', 'N/A')}")
        print(f"  Links count: {len(result.get('links', []))}")
        
        if result.get('content'):
            content_preview = result['content'][:200] + "..." if len(result['content']) > 200 else result['content']
            print(f"  Content preview: {content_preview}")
        
        print("✅ Web extract test passed\n")
        return True
        
    except Exception as e:
        print(f"❌ Web extract test failed: {e}\n")
        return False


async def test_url_validation():
    """Test URL validation in web extract."""
    print("=== Testing URL Validation ===")
    
    client = AsyncBridgeClient()
    
    # Test invalid URLs
    invalid_urls = [
        "file:///etc/passwd",
        "ftp://example.com",
        "http://localhost:8080",
        "http://127.0.0.1",
        "",
        "not-a-url"
    ]
    
    validation_passed = 0
    validation_total = len(invalid_urls)
    
    for url in invalid_urls:
        try:
            print(f"Testing invalid URL: {url}")
            result = await client.skills_web_extract(url=url)
            print(f"  ❌ Should have failed but succeeded")
        except Exception as e:
            if "validation failed" in str(e).lower() or "invalid url" in str(e).lower():
                print(f"  ✅ Correctly rejected: {str(e)[:100]}...")
                validation_passed += 1
            else:
                print(f"  ⚠️  Failed with unexpected error: {str(e)[:100]}...")
    
    print(f"URL validation: {validation_passed}/{validation_total} passed\n")
    return validation_passed == validation_total


async def main():
    """Run all web skills tests."""
    print("Phase 2 Day 1: Web Skills Testing\n")
    
    # Check if bridge socket exists
    socket_path = "/run/armorclaw/bridge.sock"
    if not os.path.exists(socket_path):
        print(f"❌ Bridge socket not found at {socket_path}")
        print("   Make sure the bridge is running before testing")
        return 1
    
    tests = [
        ("Web Search", test_web_search),
        ("Web Extract", test_web_extract),
        ("URL Validation", test_url_validation)
    ]
    
    results = []
    
    for test_name, test_func in tests:
        print(f"Running {test_name} test...")
        try:
            result = await test_func()
            results.append((test_name, result))
        except Exception as e:
            print(f"❌ {test_name} test crashed: {e}\n")
            results.append((test_name, False))
    
    # Print summary
    print("=== Test Summary ===")
    passed = sum(1 for _, result in results if result)
    total = len(results)
    
    for test_name, result in results:
        status = "✅ PASS" if result else "❌ FAIL"
        print(f"  {status} {test_name}")
    
    print(f"\nTotal: {passed}/{total} tests passed")
    
    if passed == total:
        print("🎉 All Phase 2 Day 1 web skills tests passed!")
        return 0
    else:
        print(f"❌ {total - passed} test(s) failed")
        return 1


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))