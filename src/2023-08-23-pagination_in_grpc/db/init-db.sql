CREATE TABLE book (
	id 					char(26)			NOT NULL,
	name 				varchar(255)	NOT NULL,
	description varchar(255)	NOT NULL,
	authors			varchar(255)	NOT NULL,
	published 	date					NOT NULL,
	pages 			smallserial		NOT NULL,
	isbn 				char(13)			NOT NULL,

	PRIMARY KEY (id)
);

INSERT INTO book (id, name, description, authors, published, pages, isbn) VALUES
('01H8EGW7P2GP694P593ZQRE4PP', 'Full-Stack Web Development with Go', 			'Go is a modern programming language with capabilities to enable high-performance app development.', 																																																									'Nanik Tolaram,Nick Glynn', '2023-02-01', 302, '9781803234199'),
('01H8EH2RM7HVFJG4HYA4XTV0R5', 'Domain-Driven Design with Golang', 				'Domain-driven design (DDD) is one of the most sought-after skills in the industry.',																																																																	'Matthew Boyle', 						'2022-12-01', 204, '9781804613450'),
('01H8EH316XA8GJMWMZ5MRCPVZG', 'Building Modern CLI Applications in Go', 	'Although graphical user interfaces (GUIs) are intuitive and user-friendly, nothing beats a command-line interface',																																																	'Marian Montagnino', 				'2023-03-01', 406, '9781804611654'),
('01H8EH3CKPT5BX263G0NGGKQCQ', 'Functional Programming in Go', 						'While Go is a multi-paradigm language that gives you the option to choose whichever paradigm works best',																																																						'Dylan Meeus', 							'2023-03-01', 248, '9781801811163'),
('01H8EH3MCT1J6ZRF9Z8B5TP7V4', 'Event-Driven Architecture in Golang',			'Event-driven architecture in Golang is an approach used to develop applications that shares state changes asynchronously, internally, and externally using messages.', 																							'Michael Stack', 						'2022-11-01', 384, '9781803238012'),
('01H8EH48FJHX0JXFAVFDXMVMGT', 'Test-Driven Development in Go',						'Experienced developers understand the importance of designing a comprehensive testing strategy to ensure efficient shipping and maintaining services in production.', 																								'Adelina Simion', 					'2023-04-01', 342, '9781803247878'),
('01H8EH4D94TJB6W0X20BZQ67DJ', 'Mastering Go',														'Mastering Go is the essential guide to putting Go to work on real production systems.',																																																															'Mihalis Tsoukalos', 				'2021-08-01', 682, '9781801079310'),
('01H8EH4JWPP906BD5HP7RR25KZ', 'Network Automation with Go',							'Go’s built-in first-class concurrency mechanisms make it an ideal choice for long-lived low-bandwidth I/O operations, which are typical requirements of network automation and network operations applications.', 		'Nicolas Leiva', 						'2023-01-01', 442, '9781800560925'),
('01H8EH4QAQQJAQXFFN1CH4BZ2Q', 'Microservices with Go',										'This book covers the key benefits and common issues of microservices, helping you understand the problems microservice architecture helps to solve, the issues it usually introduces, and the ways to tackle them.',	'Alexander Shuiskov', 			'2022-11-01',	328, '9781804617007'),
('01H8EH4VYYCS6M4BFVZ90RP7FS', 'Effective Concurrency in Go',							'The Go language has been gaining momentum due to its treatment of concurrency as a core language feature, making concurrent programming more accessible than ever.',																									'Burak Serdar', 						'2023-04-01', 212, '9781804619070'),
('01H8EH50M6W2T0BBP1NT5MPGDW', 'gRPC Go for Professionals',								'In recent years, the popularity of microservice architecture has surged, bringing forth a new set of requirements.', 																																																'Clément Jean', 						'2023-07-01', 260, '9781837638840');