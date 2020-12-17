#!/usr/bin/env ruby

version, darwin_sha256, linux_sha256 = ARGV

unless version
  puts "version is missing"
  exit!
end

unless darwin_sha256
  puts "darwin_sha256 is missing"
  exit!
end

unless linux_sha256
  puts "linux_sha256 is missing"
  exit!
end

begin
  content = File.read('tap-template')
rescue
  content = File.read('./scripts/release/tap-template')
end

unless content
  puts "template file missing"
  exit!
end

content.gsub! '<version>', version
content.gsub! '<darwin_sha256>', darwin_sha256
content.gsub! '<linux_sha256>', linux_sha256

puts "#{content}"
