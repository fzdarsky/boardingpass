require 'json'

Pod::Spec.new do |s|
  s.name           = 'CertificatePinning'
  s.version        = '1.0.0'
  s.summary        = 'Certificate pinning native module for BoardingPass'
  s.description    = 'Provides TLS certificate fetching and TOFU pinning for self-signed certificates'
  s.homepage       = 'https://github.com/fzdarsky/boardingpass'
  s.license        = 'MIT'
  s.author         = 'BoardingPass'
  s.platform       = :ios, '13.4'
  s.source         = { git: '' }
  s.source_files   = '**/*.{swift,m,h}'
  s.swift_version  = '5.9'

  s.dependency 'ExpoModulesCore'
  s.dependency 'React-Core'
  s.dependency 'React-RCTNetwork'
end
