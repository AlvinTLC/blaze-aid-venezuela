-- seed.sql — example data for BlazeAid Hub (dashboard/map development).
-- All rows use source='seed' so they can be removed before launch:
--   DELETE FROM aid_projects WHERE source='seed';  (and the other tables)
-- Idempotent: re-runnable via ON CONFLICT (source, external_id) DO NOTHING.

-- ===== aid_projects =====
INSERT INTO aid_projects (source, external_id, title, description, category, status, region, lat, lng, contact, url) VALUES
 ('seed','p-dc-1','Malla mesh comunitaria Caracas','Red mesh para restaurar conectividad en parroquias afectadas','connectivity','active','Distrito Capital',10.4806,-66.9036,'+58-212-5550101','https://example.org/mesh-ccs'),
 ('seed','p-mir-1','Reparación de acueducto Los Teques','Restauración de tubería principal de agua','water','active','Miranda',10.3400,-67.0400,'+58-212-5550102',''),
 ('seed','p-lag-1','Refugios temporales La Guaira','Montaje de refugios para familias desplazadas','shelter','active','La Guaira',10.6000,-66.9300,'+58-212-5550103',''),
 ('seed','p-zul-1','Brigada médica móvil Maracaibo','Atención primaria en zonas sin acceso','medical','active','Zulia',10.6400,-71.6100,'+58-261-5550104',''),
 ('seed','p-car-1','Logística de suministros Valencia','Centro de acopio y distribución','logistics','active','Carabobo',10.1600,-68.0000,'+58-241-5550105',''),
 ('seed','p-lar-1','Energía solar Barquisimeto','Paneles solares para centros de salud','energy','active','Lara',10.0700,-69.3200,'+58-251-5550106',''),
 ('seed','p-ara-1','Puente peatonal Maracay','Reparación de paso peatonal colapsado','infrastructure','closed','Aragua',10.2500,-67.6000,'+58-243-5550107',''),
 ('seed','p-tac-1','Conectividad rural Táchira','Enlaces inalámbricos para zonas montañosas','connectivity','active','Táchira',7.7700,-72.2200,'+58-276-5550108',''),
 ('seed','p-bol-1','Potabilización de agua Bolívar','Plantas portátiles de tratamiento','water','active','Bolívar',8.1200,-63.5500,'+58-285-5550109',''),
 ('seed','p-anz-1','Apoyo psicosocial Anzoátegui','Acompañamiento a damnificados','social','active','Anzoátegui',10.1300,-64.6800,'+58-281-5550110',''),
 ('seed','p-dc-2','Centro de datos de ayuda','Plataforma de coordinación de voluntarios','connectivity','active','Distrito Capital',10.5000,-66.9100,'',''),
 ('seed','p-mir-2','Comedores comunitarios Petare','Alimentación para 500 familias/día','food','active','Miranda',10.4900,-66.8100,'+58-212-5550112','')
ON CONFLICT (source, external_id) DO NOTHING;

-- ===== resources =====
INSERT INTO resources (source, external_id, type, name, quantity, unit, status, region, lat, lng, contact) VALUES
 ('seed','r-dc-1','water','Agua potable embotellada',5000,'L','available','Distrito Capital',10.4806,-66.9036,'+58-212-5550201'),
 ('seed','r-dc-2','medicine','Kits de primeros auxilios',300,'kit','available','Distrito Capital',10.5000,-66.9000,''),
 ('seed','r-mir-1','food','Raciones alimentarias',1200,'caja','available','Miranda',10.3400,-67.0400,''),
 ('seed','r-lag-1','shelter','Carpas familiares',150,'unidad','available','La Guaira',10.6000,-66.9300,''),
 ('seed','r-zul-1','fuel','Combustible diésel',2000,'L','depleted','Zulia',10.6400,-71.6100,''),
 ('seed','r-car-1','tools','Herramientas de rescate',80,'set','available','Carabobo',10.1600,-68.0000,''),
 ('seed','r-lar-1','energy','Generadores eléctricos',25,'unidad','available','Lara',10.0700,-69.3200,''),
 ('seed','r-ara-1','water','Tanques de almacenamiento',40,'unidad','available','Aragua',10.2500,-67.6000,''),
 ('seed','r-tac-1','connectivity','Routers satelitales',15,'unidad','available','Táchira',7.7700,-72.2200,''),
 ('seed','r-bol-1','medicine','Antibióticos',500,'caja','available','Bolívar',8.1200,-63.5500,''),
 ('seed','r-anz-1','food','Agua y víveres',800,'caja','available','Anzoátegui',10.1300,-64.6800,''),
 ('seed','r-zul-2','blankets','Cobijas',1000,'unidad','available','Zulia',10.6500,-71.6000,'')
ON CONFLICT (source, external_id) DO NOTHING;

-- ===== missing_persons (with coordinates for near-me) =====
INSERT INTO missing_persons (source, external_id, full_name, age, description, last_seen_region, lat, lng, last_seen_at, status, contact, photo_url) VALUES
 ('seed','m-dc-1','María Fernández',34,'Vista por última vez cerca de El Valle','Distrito Capital',10.4600,-66.9200, now()-interval '3 days','missing','+58-414-5550301',''),
 ('seed','m-dc-2','José Rodríguez',58,'Adulto mayor, requiere medicación','Distrito Capital',10.5100,-66.8800, now()-interval '5 days','missing','',''),
 ('seed','m-mir-1','Ana Gómez',27,'Estudiante, zona de Guarenas','Miranda',10.4700,-66.6100, now()-interval '2 days','found','',''),
 ('seed','m-lag-1','Carlos Pérez',41,'Pescador, desaparecido tras deslave','La Guaira',10.6000,-66.9300, now()-interval '6 days','missing','+58-424-5550304',''),
 ('seed','m-zul-1','Luisa Martínez',19,'Última vez en Maracaibo centro','Zulia',10.6400,-71.6100, now()-interval '1 day','missing','',''),
 ('seed','m-car-1','Pedro Sánchez',46,'Conductor de ayuda humanitaria','Carabobo',10.1600,-68.0000, now()-interval '4 days','missing','',''),
 ('seed','m-lar-1','Rosa Díaz',63,'Zona rural de Barquisimeto','Lara',10.0700,-69.3200, now()-interval '7 days','missing','',''),
 ('seed','m-ara-1','Miguel Torres',30,'Voluntario, no reporta desde el sismo','Aragua',10.2500,-67.6000, now()-interval '2 days','missing','+58-412-5550308','')
ON CONFLICT (source, external_id) DO NOTHING;

-- ===== volunteers =====
INSERT INTO volunteers (source, external_id, full_name, skills, availability, region, contact, status) VALUES
 ('seed','v-dc-1','Daniela Ríos','{medic,first-aid}','full-time','Distrito Capital','+58-414-5550401','available'),
 ('seed','v-dc-2','Andrés Blanco','{driver,logistics}','weekends','Distrito Capital','','available'),
 ('seed','v-mir-1','Gabriela León','{psychologist}','part-time','Miranda','','available'),
 ('seed','v-lag-1','Roberto Mejía','{rescue,diving}','full-time','La Guaira','+58-424-5550404','available'),
 ('seed','v-zul-1','Patricia Vargas','{nurse}','full-time','Zulia','','available'),
 ('seed','v-car-1','Héctor Castro','{engineer,electrical}','on-call','Carabobo','','available'),
 ('seed','v-lar-1','Carmen Ortiz','{translator,coordinator}','part-time','Lara','','available'),
 ('seed','v-tac-1','Luis Navarro','{telecom,networking}','full-time','Táchira','+58-416-5550408','available'),
 ('seed','v-bol-1','Sofía Méndez','{logistics,driver}','weekends','Bolívar','','available'),
 ('seed','v-anz-1','Jorge Herrera','{medic}','full-time','Anzoátegui','','available')
ON CONFLICT (source, external_id) DO NOTHING;

-- ===== events (timeline over last 7 days) =====
INSERT INTO events (entity, kind, payload, occurred_at)
SELECT
  (ARRAY['project','resource','missing','volunteer'])[1 + (g % 4)] AS entity,
  'ingest' AS kind,
  '{"source":"seed"}'::jsonb AS payload,
  now() - make_interval(days => (g % 7), hours => (g % 24)) AS occurred_at
FROM generate_series(0, 39) AS g;
